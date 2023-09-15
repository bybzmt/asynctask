package scheduler

import (
	"errors"
	"fmt"
)


func (s *Scheduler) GetJobConfig(gid ID, jname string) (c JobConfig, err error) {
	s.l.Lock()
	defer s.l.Unlock()

    g, ok := s.groups[gid]
    if !ok {
        return c, NotFound
    }

    g.l.Lock()
    defer g.l.Unlock()


    j, ok := g.jobs.all[jname]
    if !ok {
        return c, NotFound
    }

    return j.JobConfig, nil
}

func (s *Scheduler) SetJobConfig(gid ID, jname string, cfg JobConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

    g, ok := s.groups[gid]
    if !ok {
        return NotFound
    }

    g.l.Lock()
    defer g.l.Unlock()


    j, ok := g.jobs.all[jname]
    if !ok {
        return NotFound
    }

    j.JobConfig = cfg

    return nil
}


func (s *Scheduler) GetGroupConfigs() (out []GroupConfig) {
	s.l.Lock()
	defer s.l.Unlock()

    for _, g := range s.groups {
        out = append(out, g.GroupConfig)
    }

    return
}

func (s *Scheduler) SetGroupConfig(gid ID, cfg GroupConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return NotFound
	}

    g.GroupConfig = cfg

    return nil
}

func (s *Scheduler) SetRouterConfig(rid ID, cfg RouteConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

    for _, r := range s.routers {
        if r.Id == rid {
            r.RouteConfig = cfg
            r.init()

            s.routerChanged(r)

            return nil
        }
    }

    return Empty
}


func (s *Scheduler) GetRouteConfigs() (out []RouteConfig) {
	s.l.Lock()
	defer s.l.Unlock()

    for _, r := range s.routers {
        out = append(out, r.RouteConfig)
    }

    return
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

