package server

import (
	"asynctask/scheduler"
	"context"
	"encoding/json"
	"github.com/robfig/cron/v3"
	"time"
)

type cron_cmd int

const (
	cron_cmd_reload cron_cmd = 1
	cron_cmd_exit   cron_cmd = 2
)

type cronConfig struct {
	EditAt time.Time
	RunAt  time.Time
	Tasks  []cronTask
}

type cronTask struct {
	Cfg  string
	Task scheduler.Task
}

func (s *Server) CronRun(ctx context.Context) {
	s.Scheduler.Log.Debugln("[Info] Cron init")
	defer s.Scheduler.Log.Debugln("[Info] Cron close")

	for {
		var cfg cronConfig

		json.Marshal(&cfg)

		c := cron.New()

		for _, j := range cfg.Tasks {
			c.AddFunc(j.Cfg, func(t scheduler.Task) func() {
				return func() {
					err := s.Scheduler.AddTask(&t)
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
