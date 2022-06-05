package main

import (
	"fmt"
	"strings"
	"unsafe"
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

func (s *Scheduler) JobEmpty(name string) bool {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	j, ok := s.jobs.all[name]
	if ok {
		len := j.Tasks.Len()
		s.WaitNum -= len
		j.Tasks.Init()
	}

	return true
}

func (s *Scheduler) taskCancel(id string) bool {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	for t, _ := range s.tasks {
		_id := fmt.Sprintf("%x", unsafe.Pointer(t))
		if id == _id {
			t.worker.Cancel()
		}
	}

	return true
}

func (s *Scheduler) Status() *Statistics {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	return s.getStatData()
}

func (s *Scheduler) Close() {
	s.cmd <- CMD_CLOSE
}
