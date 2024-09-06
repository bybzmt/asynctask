package server

import (
	"github.com/robfig/cron/v3"
	"log/slog"
)

const corn_cfg_key = "cron.cfg"

type CronTask struct {
	Cfg     string
	Note    string `json:",omitempty"`
	Task    Task
	Disable bool `json:",omitempty"`
}

func (s *Server) cronConfig(c *CronTask) error {
	_, err := cron.ParseStandard(c.Cfg)
	if err != nil {
		return err
	}

	return nil
}

type cronLogger struct {
	l *slog.Logger
}

func (l *cronLogger) Info(msg string, keysAndValues ...any) {
	l.l.Debug(msg, keysAndValues...)
}

func (l *cronLogger) Error(err error, msg string, keysAndValues ...any) {
	l.l.Error(msg, append([]any{"err", err}, keysAndValues...)...)
}

func (s *Server) CronRun() {
	l := s.log.With("tag", "cron")

	l.Debug("Cron init")
	defer l.Debug("Cron close")

	c := cron.New(cron.WithLogger(&cronLogger{l}))

	s.l.Lock()

	ctx := s.ctx

	for _, j := range s.cfg.Crons {
		if j.Disable {
			continue
		}

		l.Debug("AddCron", "cfg", j.Cfg)

		_, err := c.AddFunc(j.Cfg, func(t Task) func() {
			return func() {
			    l.Info("Cron AddTask", "task", json_encode(t))

				err := s.TaskAdd(&t)
				if err != nil {
					l.Error("Cron AddTask", err)
				}
			}
		}(j.Task))

		if err != nil {
			l.Error("AddCron", "err", err)
		}
	}

	s.l.Unlock()

	c.Start()

	<-ctx.Done()

	<-c.Stop().Done()
}
