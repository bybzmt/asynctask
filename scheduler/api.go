package scheduler

import (
	"errors"
	"net/url"
	"regexp"
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

func (s *Scheduler) SetJobConfig(c *JobConfig) error {
	s.l.Lock()
	defer s.l.Unlock()

	j, ok := s.jobs[c.Name]
	if !ok {
		return NotFound
	}

	j.JobConfig = *c

	j.CmdEnv = copyMap(c.CmdEnv)
	j.HttpHeader = copyMap(c.HttpHeader)

	s.db_Job_save(c)

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

func (s *Scheduler) Groups() (out []GroupConfig) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		out = append(out, g.GroupConfig)
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

	return s.db_group_save(g)
}

func (s *Scheduler) DelGroup(gid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	g, ok := s.groups[gid]
	if !ok {
		return NotFound
	}

	if g.Id == 1 {
		return errors.New("Group Id:1 Can not Del")
	}

	g.cancel()
	delete(s.groups, gid)

	return nil
}

func (s *Scheduler) SetRoutes(pattern []string) error {
	s.router.l.Lock()
	defer s.router.l.Unlock()

	if len(pattern) == 0 {
		return errors.New("pattern empty")
	}

	var exps []*regexp.Regexp

	for _, p := range pattern {
		if p == "" {
			return errors.New("pattern empty")
		}

		exp, err := regexp.Compile(p)
		if err != nil {
			return err
		}

		exps = append(exps, exp)
	}

	s.router.routes = pattern
	s.router.exps = exps

	return nil
}

func (s *Scheduler) Routes() []string {
	s.router.l.Lock()
	defer s.router.l.Unlock()

	return s.router.routes
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
	o := new(order)
	o.Task = *t

	if (o.Task.Url == "" && o.Task.Cmd == "") || (o.Task.Url != "" && o.Task.Cmd != "") {
		return TaskError
	}

	var str string
	var scheme string

	if o.Task.Cmd != "" {
		str = strings.TrimSpace(o.Task.Cmd)
		scheme = "http"
	} else {
		str = strings.TrimSpace(o.Task.Url)
		scheme = "cli"
	}

	u, err := url.Parse(str)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		u.Scheme = scheme
	}

	u.RawQuery = ""
	u.Fragment = ""

	path := u.String()

	s.l.Lock()
	defer s.l.Unlock()

	job := s.router.Route(path)
	job = strings.Trim(job, "/")

	if job == "" {
		return errors.New("Task Not Allow")
	}

	o.Job = job

	s.TaskNextId++

	o.Id = s.TaskNextId
	o.AddTime = uint(s.now.Unix())

	if o.Task.Timer > o.AddTime {
		return s.timerAddTask(o)
	}

	return s.addTask(o)
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

func (s *Scheduler) Running() bool {
	s.l.Lock()
	defer s.l.Unlock()

	return s.running
}

func (s *Scheduler) RuleAdd(r Rule) error {
	s.l.Lock()
	defer s.l.Unlock()

	if r.Id == 0 {
		return errors.New("Id == 0")
	}

	for _, x := range s.rules {
		if x.Id == r.Id {
			return errors.New("Id duplicate")
		}
	}

	err := s.db_rule_save(r)
	if err != nil {
		return err
	}

	r.CmdEnv = copyMap(r.CmdEnv)
	r.HttpHeader = copyMap(r.HttpHeader)
	r.init()

	s.rules = append(s.rules, r)

	s.rulesSort()
	s.rulesChanged()

	return nil
}

func (s *Scheduler) RuleDel(id ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	rs := make([]Rule, 0, len(s.rules))

	for _, r := range s.rules {
		if r.Id != id {
			rs = append(rs, r)
		}
	}

	if len(rs) == len(s.rules) {
		return NotFound
	}

	s.rules = rs

	s.rulesSort()
	s.rulesChanged()

	return nil
}

func (s *Scheduler) RuleSet(cfg Rule) error {
	s.l.Lock()
	defer s.l.Unlock()

	_, ok := s.groups[cfg.GroupId]
	if !ok {
		return errors.New("Group Not Found")
	}

	for _, r := range s.rules {
		if r.Id == cfg.Id {
			r = cfg
			r.CmdEnv = copyMap(cfg.CmdEnv)
			r.HttpHeader = copyMap(cfg.HttpHeader)
			r.init()

			s.rulesSort()
			s.rulesChanged()

			return nil
		}
	}

	return Empty
}

func (s *Scheduler) Rules() (out []Rule) {
	s.l.Lock()
	defer s.l.Unlock()

    out = make([]Rule, 0, len(s.rules))

	for _, r := range s.rules {
		t := r
		t.CmdEnv = copyMap(r.CmdEnv)
		t.HttpHeader = copyMap(r.HttpHeader)

		out = append(out, t)
	}

	return
}
