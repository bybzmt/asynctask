package scheduler

import (
	"errors"
	"strings"
	"time"
)

func (s *Scheduler) GetJobConfig(jname string) (c JobConfig, err error) {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[jname]
	if !ok {
		return c, NotFound
	}

    jt.l.Lock()
    defer jt.l.Unlock()

	return jt.JobConfig, nil
}

func (s *Scheduler) SetJobConfig(jname string, cfg JobConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[jname]
	if !ok {
		return NotFound
	}

    jt.l.Lock()
    defer jt.l.Unlock()

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

	if len(s.groups) == 1 {
		return errors.New("Last Group Can not Del")
	}

	for _, r := range s.routes {
		if gid == r.GroupId {
			return errors.New("Group Use In Route")
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

func (s *Scheduler) AddRoute() (RouteConfig, error) {
	s.l.Lock()
	defer s.l.Unlock()

	r, err := s.addRoute()
	if err != nil {
		return RouteConfig{}, err
	}
	return r.RouteConfig, err
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

func (s *Scheduler) SetRouteConfig(cfg RouteConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routes {
		if r.Id == cfg.Id {
			r.RouteConfig = cfg

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

func (s *Scheduler) GetRouteConfig(id ID) (RouteConfig, error) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routes {
		if r.Id == id {
			return r.RouteConfig, nil
		}
	}

	return RouteConfig{}, NotFound
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
		return NotFound
	}

	return jt.delAllTask()
}

func (s *Scheduler) DelTask(jname string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[jname]
	if !ok {
		return NotFound
	}

	return jt.delTask(tid)
}

func (s *Scheduler) GetStatData() Statistics {
	s.l.Lock()
	defer s.l.Unlock()

	var out Statistics
	out.schedulerConfig = s.schedulerConfig
	out.Timed = s.timerTaskNum()

	out.Runs = make([]RunTaskStat, 0, s.WorkerNum)
	out.Groups = make([]GroupStat, 0, len(s.groups))
	out.Tasks = make([]TaskStat, 0, len(s.jobTask))

	for _, jt := range s.jobTask {
		useTime := 0
		if len(jt.useTimeStat.data) > 0 {
			useTime = int(jt.useTimeStat.getAll() / int64(len(jt.useTimeStat.data)) / int64(time.Millisecond))
		}

		sec := 0

		sec2 := jt.lastTime.Unix()
		if sec2 > 0 {
			sec = int(s.now.Sub(jt.lastTime) / time.Second)
		}

		tmp := TaskStat{
			JobConfig: jt.JobConfig,
			Name:      jt.name,
			NowNum:    int(jt.nowNum.Load()),
			RunNum:    int(jt.runNum.Load()),
			OldNum:    int(jt.oldNum.Load()),
			ErrNum:    int(jt.errNum.Load()),
			WaitNum:   int(jt.waitNum.Load()),
			UseTime:   useTime,
			LastTime:  sec,
			GroupId:   jt.group.Id,
		}

		if jt.group != nil {
			jt.group.l.Lock()

			if j, ok := jt.group.jobs.all[jt.name]; ok {
				tmp.Load = j.loadStat.getAll()
				tmp.Score = j.score
			}

			jt.group.l.Unlock()
		}

		out.RunNum += tmp.RunNum
		out.ErrNum += tmp.ErrNum
		out.WaitNum += tmp.WaitNum
		out.OldNum += tmp.OldNum
		out.Tasks = append(out.Tasks, tmp)
	}

	for _, s := range s.groups {
		group, runs := s.getStatData()

		out.Capacity += group.Capacity
		out.Load += group.Load
		out.NowNum += group.NowNum
		out.WorkerNum += group.WorkerNum
		out.Groups = append(out.Groups, group)
		out.Runs = append(out.Runs, runs...)
	}

	return out
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
