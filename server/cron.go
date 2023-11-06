package server

import (
	"asynctask/scheduler"
	"context"
	"encoding/json"
	"github.com/robfig/cron/v3"
	bolt "go.etcd.io/bbolt"
	"net/http"
	"time"
)

type cron_cmd int

const (
	cron_cmd_reload cron_cmd = 1
)

const corn_cfg_key = "cron.cfg"

type cronConfig struct {
	EditAt int64
	RunAt  int64
	Tasks  []cronTask
}

type cronTask struct {
	Id   int
	Cfg  string
	Note string
	Task scheduler.Task
}

func (s *Server) initCron() error {
	s.cronCmd = make(chan cron_cmd)

	return nil
}

func (s *Server) CronRun(ctx context.Context) {
	s.Scheduler.Log.Debugln("Cron init")
	defer s.Scheduler.Log.Debugln("Cron close")

	for {
		cfg := s.cron_run_cfg()

		c := cron.New()

		for _, j := range cfg.Tasks {
			c.AddFunc(j.Cfg, func(t scheduler.Task) func() {
				return func() {
					err := s.Scheduler.TaskAdd(t)
					if err != nil {
						s.Scheduler.Log.Errorln("Cron AddTask", err)
					}
				}
			}(j.Task))
		}

		c.Start()

		select {
		case <-ctx.Done():
			<-c.Stop().Done()

			return
		case ctx := <-s.cronCmd:
			_ = ctx

			<-c.Stop().Done()
		}
	}
}

func (s *Server) cron_run_cfg() cronConfig {

	var cfg cronConfig

	//key: config/cron.cfg
	err := s.Scheduler.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config")
		if err != nil {
			return err
		}

		val := bucket.Get([]byte(corn_cfg_key))
		if val == nil {
			return nil
		}

		err = json.Unmarshal(val, &cfg)
		if err != nil {
			return err
		}

		cfg.RunAt = time.Now().Unix()

		val, err = json.Marshal(&cfg)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(corn_cfg_key), val)
	})

	if err != nil {
		s.Scheduler.Log.Warnln("cron_run_cfg", err)
	}

	return cfg
}

func (s *Server) page_cron_getConfig(r *http.Request) any {
	var cfg cronConfig

	//key: config/cron.cfg
	err := s.Scheduler.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "config")
		if bucket == nil {
			return nil
		}

		val := bucket.Get([]byte(corn_cfg_key))

		if val == nil {
			return nil
		}

		return json.Unmarshal(val, &cfg)
	})

	if err != nil {
		s.Scheduler.Log.Warnln("cron_run_cfg", err)
	}

	return cfg
}

func (s *Server) page_cron_setConfig(r *http.Request) any {

	var tasks []cronTask

	if err := httpReadJson(r, &tasks); err != nil {
		return err
	}

	for _, j := range tasks {
		_, err := cron.ParseStandard(j.Cfg)
		if err != nil {
			return err
		}

		_, err = s.Scheduler.TaskCheck(j.Task)
		if err != nil {
			return err
		}
	}

	var cfg cronConfig
	cfg.Tasks = tasks
	cfg.EditAt = time.Now().Unix()

	//key: config/cron.cfg
	err := s.Scheduler.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config")
		if err != nil {
			return err
		}

		val, err := json.Marshal(&cfg)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(corn_cfg_key), val)
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Server) page_cron_reload(r *http.Request) any {

	s.cronCmd <- cron_cmd_reload

	return nil
}
