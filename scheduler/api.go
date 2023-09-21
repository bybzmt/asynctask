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

	g.l.Lock()
	defer g.l.Unlock()

	cfg.Id = g.Id
	g.GroupConfig = cfg

	return s.saveGroup(g)
}

func (s *Scheduler) DelGroup(gid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return NotFound
	}

	if len(s.groups) == 1 {
		return errors.New("Last Group Can not Del")
	}

	for _, r := range s.routes {
		for _, id := range r.Groups {
			if id == gid {
				return errors.New("Group Use In Route")
			}
		}
	}

	g.l.Lock()
	defer g.l.Unlock()

	if len(g.jobs.all) > 0 {
		return errors.New("Jobs Not Empty")
	}

	g.close()
	delete(s.groups, gid)

	return nil
}

func (s *Scheduler) AddRoute() (ID, error) {
	r, err := s.addRoute()
	if err != nil {
		return 0, err
	}
	return r.Id, err
}

func (s *Scheduler) DelRoute(rid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	if len(s.routes) == 1 {
		return errors.New("Last Route Can not Del")
	}

	rs := make([]*router, 0, len(s.routes))

	for _, r := range s.routes {
		if r.Id != rid {
			rs = append(rs, r)
		}
	}

	if len(rs) == len(s.routes) {
		return NotFound
	}

	s.routes = rs

	return nil
}

func (s *Scheduler) SetRouteConfig(rid ID, cfg RouteConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routes {
		if r.Id == rid {
			cfg.Id = r.Id
			r.RouteConfig = cfg

			if err := r.init(); err != nil {
				return err
			}

			s.routersSort()
			s.routeChanged(r)

			return nil
		}
	}

	return Empty
}

func (s *Scheduler) GetRouteConfigs() (out []RouteConfig) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routes {
		out = append(out, r.RouteConfig)
	}

	return
}

func (s *Scheduler) OrderCancel(gid, oid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return NotFound
	}

	g.l.Lock()
	defer g.l.Unlock()

	for t := range g.orders {
		if t.Id == oid {
			t.worker.Cancel()
			return nil
		}
	}

	return NotFound
}

func (s *Scheduler) JobDelIdle(gid ID, jname string) error {
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

	if j.mode != job_mode_idle {
		return NotFound
	}

	g.jobs.removeJob(j)
	g.jobs.idleLen--

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

func (s *Scheduler) DelTask(jname string, tid ID) error {
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
