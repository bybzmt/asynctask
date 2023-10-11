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

	jt.group.l.Lock()
	defer jt.group.l.Unlock()

	return jt.JobConfig, nil
}

func (s *Scheduler) SetJobConfig(jname string, cfg JobConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

	jt.group.l.Lock()
	defer jt.group.l.Unlock()

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

func (s *Scheduler) GetGroupConfigs() (out []GroupConfig) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		g.l.Lock()
		out = append(out, g.GroupConfig)
		g.l.Unlock()
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

	g.l.Lock()
	defer g.l.Unlock()

	return g.GroupConfig, nil
}

func (s *Scheduler) SetGroupConfig(cfg GroupConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[cfg.Id]
	if !ok {
		return NotFound
	}

	g.l.Lock()
	defer g.l.Unlock()

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

	g.l.Lock()
	defer g.l.Unlock()

	g.close()
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

    jt.group.l.Lock()
    defer jt.group.l.Unlock()

	return jt.delAllTask()
}

func (s *Scheduler) DelTask(jname string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

    jt.group.l.Lock()
    defer jt.group.l.Unlock()

	return jt.delTask(tid)
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

func (s *Scheduler) GetStatData() Statistics {
	s.l.Lock()
	defer s.l.Unlock()

	var out Statistics
	out.schedulerConfig = s.schedulerConfig
	out.Timed = s.timerTaskNum()

	out.Runs = make([]RunTaskStat, 0, s.WorkerNum)
	out.Groups = make([]GroupStat, 0, len(s.groups))
	out.Tasks = make([]JobStat, 0, len(s.jobs))

	for _, jt := range s.jobs {
        tmp := jt.group.getJobStat(jt)

		out.RunNum += tmp.RunNum
		out.ErrNum += tmp.ErrNum
		out.WaitNum += tmp.WaitNum
		out.OldNum += tmp.OldNum
		out.Tasks = append(out.Tasks, tmp)
	}

	for _, s := range s.groups {
		group := s.getGroupStat()
		runs := s.getRunTaskStat()

		out.Capacity += group.Capacity
		out.Load += group.Load
		out.NowNum += group.NowNum
		out.WorkerNum += group.WorkerNum
		out.Groups = append(out.Groups, group)

		out.Runs = append(out.Runs, runs...)
	}

	return out
}

