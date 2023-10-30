package server

import (
	"asynctask/scheduler"
	"context"
	bolt "go.etcd.io/bbolt"
	"net"
	"net/http"
	"time"
	_ "time/tzdata"
)

type Server struct {
	Scheduler  scheduler.Scheduler
	Http       http.Server
	Redis      RedisConfig
	DbFile     string
	HttpEnable bool
	cronCmd    chan cron_cmd
}

func (s *Server) initDb() error {
	if s.Scheduler.Db == nil {
		db, err := bolt.Open(s.DbFile, 0666, nil)
		if err != nil {
			return err
		}

		s.Scheduler.Db = db
	}
	return nil
}

func (s *Server) Init() error {

	if err := s.initDb(); err != nil {
		return err
	}

	if err := s.Scheduler.Init(); err != nil {
		return err
	}

	if err := s.initCron(); err != nil {
		return err
	}

	if err := s.initRedis(); err != nil {
		return err
	}

	s.initHttp()

	return nil
}

func (s *Server) Run(pctx context.Context) {

	ctx, canceler := context.WithCancel(pctx)
	defer canceler()

	go func() {
		time.Sleep(time.Millisecond * 10)

		exit := make(chan int, 3)

		go func() {
			if s.HttpEnable {
				s.Http.BaseContext = func(net.Listener) context.Context {

					go func() {
						<-ctx.Done()
						s.Http.Close()
					}()

					return ctx
				}

				err := s.Http.ListenAndServe()

				if err != http.ErrServerClosed {
					s.Scheduler.Log.Warnln(err)

					canceler()
				}
			}

			exit <- 1
		}()

		go func() {
			if s.Redis.Key != "" {
				s.RedisRun(ctx)
			}

			exit <- 1
		}()

		go func() {
			s.CronRun(ctx)
			exit <- 1
		}()

		<-exit
		<-exit
		<-exit

		s.Scheduler.Close()
	}()

	s.Scheduler.Run()
}
