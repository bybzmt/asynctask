package scheduler

import (
	"errors"
	"strings"
)

func (s *Scheduler) GetJobConfig(jname string) (c JobConfig, err error) {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return c, NotFound
	}

	return jt.JobConfig, nil
}

func (s *Scheduler) SetJobConfig(jname string, cfg JobConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

	jt.JobConfig = cfg

	return nil
}

func (s *Scheduler) AddGroup() (GroupConfig, error) {
	s.l.Lock()
	defer s.l.Unlock()

	g, err := s.addGroup()
	if err != nil {
		return GroupConfig{}, err
	}

	return g.GroupConfig, nil
}

func (s *Scheduler) GetGroupStat() (out []GroupStat) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		out = append(out, g.getGroupStat())
	}

	return
}

func (s *Scheduler) GetGroupConfig(id ID) (GroupConfig, error) {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[id]
	if !ok {
		return GroupConfig{}, NotFound
	}

	return g.GroupConfig, nil
}

func (s *Scheduler) SetGroupConfig(cfg GroupConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[cfg.Id]
	if !ok {
		return NotFound
	}

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

	for _, r := range s.routes {
		if gid == r.GroupId {
			return errors.New("Group Use In Route")
		}
	}

	if len(s.groups) == 1 {
		return errors.New("Last Group Can not Del")
	}

	g.cancel()
	delete(s.groups, gid)

	return nil
}

func (s *Scheduler) AddRoute() (TaskConfig, error) {
	s.l.Lock()
	defer s.l.Unlock()

	r, err := s.addRoute()
	if err != nil {
		return TaskConfig{}, err
	}
	return r.TaskConfig, err
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

func (s *Scheduler) SetRouteConfig(cfg TaskConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	_, ok := s.groups[cfg.GroupId]
	if !ok {
		return errors.New("Group Not Found")
	}

	for _, r := range s.routes {
		if r.Id == cfg.Id {
			r.TaskConfig = cfg
			r.TaskBase = copyTaskBase(cfg.TaskBase)

			if err := r.init(); err != nil {
				return err
			}

			s.routersSort()
			s.routeChanged()

			return nil
		}
	}

	return Empty
}

func (s *Scheduler) GetRouteConfig(id ID) (TaskConfig, error) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routes {
		if r.Id == id {
			return r.TaskConfig, nil
		}
	}

	return TaskConfig{}, NotFound
}

func (s *Scheduler) GetRouteConfigs() (out []TaskConfig) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routes {
		out = append(out, r.TaskConfig)
	}

	return
}

func (s *Scheduler) TaskCancel(oid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	for t := range s.orders {
		if t.Id == oid {
			t.cancel()
			return nil
		}
	}

	return NotFound
}

func (s *Scheduler) JobDelIdle(name string) error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.delIdleJob(name)
}

func (s *Scheduler) JobEmpty(jname string) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

	return jt.delAllTask()
}

func (s *Scheduler) DelTask(jname string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

	return jt.delTask(tid)
}

func (s *Scheduler) TaskAdd(t *Task) error {
	t.Name = strings.TrimSpace(t.Name)
	t.Url = strings.TrimSpace(t.Url)
	t.Cmd = strings.TrimSpace(t.Cmd)

	if (t.Url == "" && t.Cmd == "") || (t.Url != "" && t.Cmd != "") {
		return TaskError
	}

	if t.Cmd != "" {
		if t.Name == "" {
			t.Name = t.Cmd
		}
	} else {
		if t.Name == "" {
			t.Name = t.Url
		}
	}

	s.l.Lock()
	defer s.l.Unlock()

	s.TaskNextId++
	t.Id = uint(s.TaskNextId)
	t.AddTime = uint(s.now.Unix())

	if t.Timer > uint(s.now.Unix()) {
		return s.timerAddTask(t)
	}

	return s.addTask(t)
}

func (s *Scheduler) GetStatData() Statistics {
	s.l.Lock()
	defer s.l.Unlock()

	var out Statistics
	out.schedulerConfig = s.schedulerConfig
	out.Timed = s.timedNum

	out.Groups = make([]GroupStat, 0, len(s.groups))
	out.Tasks = make([]JobStat, 0, len(s.jobs))

	for _, jt := range s.jobs {
		tmp := jt.group.getJobStat(jt)

		out.Tasks = append(out.Tasks, tmp)
	}

	for _, s := range s.groups {
		group := s.getGroupStat()

		out.Groups = append(out.Groups, group)
	}

	return out
}

func (s *Scheduler) GetRunTaskStat() []RunTaskStat {
	s.l.Lock()
	defer s.l.Unlock()

    return s.getRunTaskStat()
}
