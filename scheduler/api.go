package scheduler

import (
	"errors"
	"fmt"
	"strings"
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

func (s *Scheduler) SetRouteConfig(rid ID, cfg RouteConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

    for _, r := range s.routers {
        if r.Id == rid {
            r.RouteConfig = cfg
            err := r.init()
            if err != nil {
                return err
            }

            s.routeChanged(r)

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
		return errors.New(fmt.Sprintf("job:%s not found", jname))
	}

    return jt.delAllTask()
}

func (s *Scheduler) DelOrder(jname string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[jname]
	if !ok {
		return errors.New(fmt.Sprintf("job:%s not found", jname))
	}

    return jt.delTask(tid)
}

func (s *Scheduler) GetStatData() (out []*Statistics) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, s := range s.groups {
		out = append(out, s.getStatData())
	}

	return
}

func (s *Scheduler) AddTask(t *Task) error {
	t.Name = strings.TrimSpace(t.Name)

	if t.Name == "" {
		return TaskError
	}

	if t.Http == nil && t.Cli == nil {
		return TaskError
	}

	s.l.Lock()
	defer s.l.Unlock()

	s.TaskNextId++
	t.Id = uint(s.TaskNextId)
	t.AddTime = uint(s.now.Unix())

	if t.Trigger > uint(s.now.Unix()) {
		return s.timerAddTask(t)
	}

	return s.addTask(t)
}
