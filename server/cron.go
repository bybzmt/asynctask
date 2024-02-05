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
	s.log.Debugln("Cron init")
	defer s.log.Debugln("Cron close")

	c := cron.New()

	s.l.Lock()

	ctx := s.ctx

	for _, j := range s.cfg.Crons {
		if j.Disable {
			continue
		}

		c.AddFunc(j.Cfg, func(t Task) func() {
			return func() {
				err := s.TaskAdd(&t)
				if err != nil {
					s.log.Errorln("Cron AddTask", err)
				}
			}
		}(j.Task))
	}

	s.l.Unlock()

	c.Start()

	<-ctx.Done()

	<-c.Stop().Done()
}
