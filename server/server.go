package server

import (
	"asynctask/scheduler"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	bolt "go.etcd.io/bbolt"

	"github.com/sirupsen/logrus"
)

type debugLog struct {
	log logrus.FieldLogger
}

func (l *debugLog) Println(val ...interface{}) {
	l.log.Debugln(val...)
}

type Server struct {
	l sync.Mutex

	cfg Config

	router router

	s    *scheduler.Scheduler
	http *http.Server
	log  logrus.FieldLogger
	db   *bolt.DB

	config string
	dbFile string
	now    time.Time

	ctx    context.Context
	cancel context.CancelFunc

	run   int
	timer timer
}

func (s *Server) getConfig() (*Config, error) {

	c := new(Config)

	f, err := os.Open(s.config)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := json.NewDecoder(f)

	err = d.Decode(c)
	if err != nil {
		return nil, err
	}

	cc := s.getSchedulerConfig(c)

	if err = s.s.CheckConfig(cc); err != nil {
		return nil, err
	}

	c.Group = cc.Group
	c.WorkerNum = cc.WorkerNum
	c.Parallel = cc.Parallel
	c.JobsMaxIdle = cc.JobsMaxIdle

	if c.HttpAddr == "" {
		c.HttpAddr = "127.0.0.1:80"
	}

	if c.CloseWait == 0 {
		c.CloseWait = 10
	}

	if c.Dirver == nil {
		c.Dirver = map[string]*Dirver{}
	}

	var router router
	err = router.set(c.Routes)
	if err != nil {
		return nil, err
	}

	for _, r := range c.Routes {
		if strings.ToLower(r.Dirver) == "http" || r.Dirver == "" {
			r.Dirver = "http"

			if _, ok := c.Dirver["http"]; !ok {
				c.Dirver["http"] = &Dirver{Type: DIRVER_HTTP}
			}
		}

		if _, ok := c.Dirver[r.Dirver]; !ok {
			return nil, fmt.Errorf("undefined dirver:%s", r.Dirver)
		}
	}

	for _, x := range c.Dirver {
		if err := x.init(s); err != nil {
			return nil, err
		}
	}

	if _, _, err = net.SplitHostPort(c.HttpAddr); err != nil {
		return nil, err
	}

	for _, r := range c.Redis {
		if err := r.checkConfig(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (s *Server) initDb() error {
	db, err := bolt.Open(s.dbFile, 0666, &bolt.Options{
		NoSync: true,
	})
	if err != nil {
		return err
	}

	s.db = db

	return nil
}

func (s *Server) getSchedulerConfig(cfg *Config) *scheduler.Config {
	c := &scheduler.Config{
		Group:       cfg.Group,
		WorkerNum:   cfg.WorkerNum,
		Parallel:    cfg.Parallel,
		JobsMaxIdle: cfg.JobsMaxIdle,
		Jobs:        cfg.Jobs,
		Groups:      cfg.Groups,
		Dirver:      scheduler.DirverFunc(s.dirver),
		Log:         &debugLog{log: s.log},
	}

	return c
}

func (s *Server) initScheduler() error {
	c := s.getSchedulerConfig(&s.cfg)

	x, err := scheduler.New(c)

	if err != nil {
		return err
	}

	s.s = x

	return nil
}

func New(config, db string, log logrus.FieldLogger) (*Server, error) {

	s := Server{
		config: config,
		dbFile: db,
		log:    log,
	}

	if s.log == nil {
		s.log = logrus.StandardLogger()
	}

	cfg, err := s.getConfig()
	if err != nil {
		return nil, err
	}
	s.cfg = *cfg

	if err := s.initDb(); err != nil {
		return nil, err
	}

	if err := s.initScheduler(); err != nil {
		return nil, err
	}

	s.timer.init()

	err = s.router.set(s.cfg.Routes)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Server) Start() {
	s.store_init()

	s.run = 1
	s.now = time.Now()

	go func() {
		tick := time.Tick(time.Second)

		for s.run == 1 {
			now := <-tick
			s.now = now

			s.checkTimer(now)

			if err := s.db.Sync(); err != nil {
				s.log.Error("db Sync", err)
			}
		}
	}()

	go func() {
		for s.run == 1 {
            s.log.Debugln("start")

			s.l.Lock()
			s.ctx, s.cancel = context.WithCancel(context.Background())
			ctx := s.ctx

			if s.cfg.HttpEnable {
				s.initHttp()

				go func() {
					err := s.http.ListenAndServe()

					if err != http.ErrServerClosed {
						s.log.Warnln(err)
					}
				}()
			}

			for _, r := range s.cfg.Redis {
				go r.RedisRun(s)
			}

			go s.CronRun()

			s.l.Unlock()

			<-ctx.Done()
		}
	}()

	s.s.Start()

	if err := s.db.Sync(); err != nil {
		s.log.Error("db Sync", err)
	}

	if err := s.db.Close(); err != nil {
		s.log.Error("db Close", err)
	}

	s.run = 3
}

func (s *Server) Reload() error {
	s.l.Lock()
	defer s.l.Unlock()

	cfg, err := s.getConfig()
	if err != nil {
		return err
	}

	s.cfg = *cfg

	err = s.reload()
	if err != nil {
		s.log.Error("reload internal error", err)
		return err
	}

	return nil
}

func (s *Server) reload() error {
	if s.http != nil {
		ctx, fn := context.WithTimeout(context.Background(), time.Second*2)
		defer fn()

		s.http.Shutdown(ctx)
	}

	if s.cancel != nil {
		s.cancel()
	}

	err := s.router.set(s.cfg.Routes)
	if err != nil {
		return err
	}

	err = s.s.SetConfig(s.getSchedulerConfig(&s.cfg))
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() {
	s.l.Lock()
	defer s.l.Unlock()

	s.run = 2

	s.reload()
	s.s.Stop()
}

func (s *Server) Kill() {
	s.Stop()

	s.s.Kill()
}

func (s *Server) WaitStop() {
	for n := 0; s.run < 3 && n < int(s.cfg.CloseWait); n++ {
		time.Sleep(time.Millisecond * 10)
	}
}
