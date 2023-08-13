package scheduler

import (
	"errors"
	"fmt"
)

func (s *Scheduler) JobEmpty(gid, jid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("scheduler:%d not found", gid))
	}

	return g.jobs.jobEmpty(jid)
}

func (s *Scheduler) JobDelIdle(gid, jid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("scheduler:%d not found", gid))
	}

	return g.jobs.jobDelIdle(jid)
}

func (s *Scheduler) JobPriority(gid, jid ID, priority int) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("scheduler:%d not found", gid))
	}

	return g.jobs.jobPriority(jid, priority)
}

func (s *Scheduler) JobParallel(gid, jid ID, parallel uint32) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("scheduler:%d not found", gid))
	}

	return g.jobs.jobParallel(jid, parallel)
}

func (s *Scheduler) OrderCancel(gid, oid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("scheduler:%d not found", gid))
	}

	for t := range g.orders {
		if t.Id == oid {
			t.worker.Cancel()
		}
	}

	return nil
}

func (s *Scheduler) DelOrder(gid, jid, oid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("group:%d not found", gid))
	}

    g.l.Lock()
    defer g.l.Unlock()

    j := g.jobs.getJob(jid)
    if j == nil {
		return errors.New(fmt.Sprintf("job:%d not found", jid))
    }

    return j.delOrder(oid)
}

func (s *Scheduler) GetStatData() (out []*Statistics) {
	s.l.Lock()
	defer s.l.Lock()

	for _, s := range s.groups {
		out = append(out, s.getStatData())
	}

	return
}
