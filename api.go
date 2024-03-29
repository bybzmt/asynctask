package main

import (
	"fmt"
	"math"
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

func (s *Scheduler) AddOrderRel(o *Order) bool {
	if !s.running {
		return false
	}

	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return false
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

		if j.mode == JOB_MODE_RUNNABLE {
			s.jobs.remove(j)
			s.jobs.idlePushBack(j)
		}
	}

	return ok
}

func (s *Scheduler) JobDelIdle(name string) bool {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	j, ok := s.jobs.all[name]
	if ok {
		if j.mode == JOB_MODE_IDLE {
			s.jobs.idleRmove(j)
			delete(s.jobs.all, j.Name)
			return true
		}
	}

	return false
}

func (s *Scheduler) JobPriority(name string, priority int) bool {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	j, ok := s.jobs.all[name]
	if ok {
		j.priority = priority
	}

	return ok
}

func (s *Scheduler) JobParallel(name string, parallel int) bool {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	parallel_abs := uint(math.Abs(float64(parallel)))

	j, ok := s.jobs.all[name]
	if ok {
		j.parallel = parallel
		j.parallel_abs = parallel_abs
		s.jobs.modeCheck(j)
	}

	return ok
}

func (s *Scheduler) taskCancel(id string) bool {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	for t, _ := range s.tasks {
		_id := fmt.Sprintf("%x", unsafe.Pointer(t))
		if id == _id {
			t.worker.Cancel()
			return true
		}
	}

	return false
}

func (s *Scheduler) Status() *Statistics {
	s.cmd <- CMD_SUSPEND
	defer func() { s.cmd <- CMD_RESUME }()

	return s.getStatData()
}

func (s *Scheduler) Close() {
	s.cmd <- CMD_CLOSE
}
