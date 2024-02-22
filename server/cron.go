package server

import (
	"github.com/robfig/cron/v3"
)

const corn_cfg_key = "cron.cfg"

type CronTask struct {
	Cfg     string
	Note    string
	Task    Task
	Disable bool
}

func (s *Server) cronConfig(c *CronTask) error {
	_, err := cron.ParseStandard(c.Cfg)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) CronRun() {
	l := s.log.WithField("tag", "cron")

	l.Debugln("Cron init")
	defer l.Debugln("Cron close")

	c := cron.New()

	s.l.Lock()

	ctx := s.ctx

	for _, j := range s.cfg.Crons {
		if j.Disable {
			continue
		}

		l.Debugln("AddCron", j.Cfg)

		_, err := c.AddFunc(j.Cfg, func(t Task) func() {
			return func() {
				l.WithField("task", json_encode(t)).Info("addTask")

				err := s.TaskAdd(&t)
				if err != nil {
					l.Errorln("Cron AddTask", err)
				}
			}
		}(j.Task))

		if err != nil {
			l.Errorln("AddCron", err)
		}
	}

	s.l.Unlock()

	c.Start()

	<-ctx.Done()

	<-c.Stop().Done()
}
