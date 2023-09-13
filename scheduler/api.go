package scheduler

import (
	"errors"
	"fmt"
)


func (s *Scheduler) GetJobConfig(gid, jid ID) (*JobConfig, error) {
    return nil, nil
}

func (s *Scheduler) SetJobConfig(gid, jid ID, cfg *JobConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

    return nil
}


func (s *Scheduler) GetGroupConfig(gid ID, cfg *GroupConfig) error {
    return nil
}

func (s *Scheduler) SetGroupConfig(gid ID, cfg GroupConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return errors.New(fmt.Sprintf("scheduler:%d not found", gid))
	}

    g.GroupConfig = cfg

    return nil
}

func (s *Scheduler) SetRouterConfig(rid ID, cfg RouterConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

    err := Empty

    for _, g := range s.routers {
        if g.id == rid {
            err = nil
            g.RouterConfig = cfg
        }
    }

    return err
}


func (s *Scheduler) GetRouterConfig(rid ID, cfg *RouterConfig) error {
    return nil
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


func (s *Scheduler) JobEmpty(jname string) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[jname]
	if !ok {
		return errors.New(fmt.Sprintf("group:%d not found", jname))
	}

    return jt.delAllTask()
}

func (s *Scheduler) DelOrder(jname string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[jname]
	if !ok {
		return errors.New(fmt.Sprintf("group:%d not found", jname))
	}

    return jt.delTask(tid)
}

func (s *Scheduler) GetStatData() (out []*Statistics) {
	s.l.Lock()
	defer s.l.Unlock()

    s.Log.Debugln("GetStatData")

	for _, s := range s.groups {
		out = append(out, s.getStatData())
	}

	return
}
