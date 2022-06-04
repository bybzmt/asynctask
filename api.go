package main

import (
	"strings"
)

func (s *Scheduler) AddOrder(o *Order) bool {
	if !s.running {
		return false
	}

	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return false
	}

	if s.redis != nil {
		return s.redis_add(o)
	}

	s.order <- o
	return true
}

func (s *Scheduler) Status() *Statistics {
	s.cmd <- CMD_SUSPEND
	t := s.getStatData()
	s.cmd <- CMD_RESUME
	return t
}

func (s *Scheduler) Close() {
	s.cmd <- CMD_CLOSE
}
