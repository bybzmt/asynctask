package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Scheduler struct {
	cfg Config

	log Logger

	l sync.Mutex

	now time.Time

	idle    *job
	idleLen int

	jobs   map[string]*job
	groups map[string]*group
	dirver Dirver

	rules rules

	orders   map[*order]struct{}
	complete chan *order

	today    int
	running  int
	statSize int

	ctx    context.Context
	cancel context.CancelFunc
}

func (s *Scheduler) init() error {

	if s.log == nil {
		s.log = new(nullLogger)
	}

	s.today = time.Now().Day()

	s.statSize = 60 * 5

	s.jobs = make(map[string]*job)
	s.complete = make(chan *order)
	s.orders = make(map[*order]struct{})

	s.idle = &job{}
	s.idle.next = s.idle
	s.idle.prev = s.idle

	s.ctx, s.cancel = context.WithCancel(context.Background())

	return nil
}

func New(c *Config) (*Scheduler, error) {
	if c == nil {
		panic("Config is nil")
	}

	s := new(Scheduler)
	s.init()

	if err := s.SetConfig(c); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Scheduler) GetConfig() *Config {
	s.l.Lock()
	defer s.l.Unlock()

	c := s.cfg
	c.Jobs = append([]*Job(nil), s.cfg.Jobs...)
	c.Groups = make(map[string]*Group)

	for k, v := range s.cfg.Groups {
		c.Groups[k] = v
	}

	return &c
}

func (s *Scheduler) CheckConfig(c *Config) error {
	if c.Group == "" {
		c.Group = "default"
	}

	if c.WorkerNum == 0 {
		c.WorkerNum = 10
	}

	if c.Parallel == 0 {
		c.Parallel = 1
	}

	if c.JobsMaxIdle == 0 {
		c.JobsMaxIdle = 100
	}

	if c.Log == nil {
		c.Log = new(nullLogger)
	}

	if c.Groups == nil {
		c.Groups = make(map[string]*Group)
	}

	if _, ok := c.Groups[c.Group]; !ok {
		c.Groups[c.Group] = &Group{
			Note:      c.Group,
			WorkerNum: c.WorkerNum,
		}
	}

	//check
	if c.Dirver == nil {
		return DirverError
	}

	for _, j := range c.Jobs {
		if j.Group == "" {
			j.Group = c.Group
		}

		_, ok := c.Groups[j.Group]
		if !ok {
			return errors.New("Job:" + j.Pattern + " Group:" + j.Group + " Not Found")
		}
	}

	var j rules

	if err := j.set(c.Jobs); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) SetConfig(c *Config) error {
	s.l.Lock()
	defer s.l.Unlock()

	//default
	if err := s.CheckConfig(c); err != nil {
		return err
	}

	var j rules

	if err := j.set(c.Jobs); err != nil {
		return err
	}

	//apply
	s.cfg = *c
	s.rules = j
	s.dirver = c.Dirver

	if c.Log != nil {
		s.log = c.Log
	} else {
		c.Log = new(nullLogger)
	}

	groups := make(map[string]*group)

	for k, g := range s.groups {
		if cg, ok := s.cfg.Groups[k]; ok {
			g.Group = *cg
			groups[k] = g
		} else {
			g.WorkerNum = 0
		}
	}

	s.groups = groups

	for k, j := range s.jobs {
		t := s.rules.match(k)
		if t != nil {
			j.priority = t.Priority
			j.parallel = t.Parallel
			j.group = t.Group
		} else {
			j.priority = 0
			j.parallel = s.cfg.Parallel
			j.group = s.cfg.Group
		}

		g := s.getGroup(j.group)

		if j.g != g {
			jobRemove(j)

			if j.g != nil {
				j.g.waitNum -= j.len()
			}

			g.waitNum += j.len()
			j.g = g

			g.runAdd(j)
		}

		g.modeCheck(j)
	}

	return nil
}

func (s *Scheduler) Start() {
	s.log.Println("scheduler start")
	defer s.log.Println("scheduler stop")

	s.running = 1
	defer func() { s.running = 3 }()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	s.now = time.Now()

	for {
		select {
		case now := <-ticker.C:
			s.onTick(now)

		case o := <-s.complete:
			s.onComplete(o)
		}

		if s.running != 1 {
			s.l.Lock()
			l := len(s.orders)
			s.l.Unlock()

			if l == 0 {
				return
			}
		}
	}
}

func (s *Scheduler) onTick(now time.Time) {
	s.l.Lock()
	defer s.l.Unlock()

	s.log.Println("tick")

	s.now = now

	s.dayCheck()
	s.statMaintain()

	empty := true

	for _, g := range s.groups {
		for g.dispatch() {
			empty = true
		}
	}

	if empty && len(s.orders) == 0 && s.cfg.OnIdle != nil {
		s.cfg.OnIdle()
	}
}

func (s *Scheduler) onComplete(o *order) {
	s.l.Lock()
	defer s.l.Unlock()

	s.now = time.Now()

	o.g.end(o)

	for o.g.dispatch() {
	}
}

func (s *Scheduler) Stop() {
	s.l.Lock()
	defer s.l.Unlock()

	if s.running != 1 {
		return
	}

	s.log.Println("Scheduler closing...")

	s.running++

	for _, g := range s.groups {
		g.WorkerNum = 0
	}
}

func (s *Scheduler) Kill() {
	s.Stop()

	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scheduler) WaitStop() {
	for s.running != 3 {
		time.Sleep(time.Millisecond * 10)
	}
}

func (s *Scheduler) delIdleJob(name string) error {
	j, ok := s.jobs[name]
	if !ok {
		return NotFound
	}

	if j.mode != job_mode_idle {
		return NotFound
	}

	jobRemove(j)

	delete(s.jobs, name)

	return nil
}

func (s *Scheduler) dayCheck() {
	today := s.now.Day()

	if s.today != today {
		s.today = today

		for _, j := range s.jobs {
			j.oldRun = j.runNum
			j.oldErr = j.errNum
			j.runNum = 0
			j.errNum = 0
		}

		for _, g := range s.groups {
			g.oldRun = g.runNum
			g.oldErr = g.errNum
			g.runNum = 0
			g.errNum = 0
		}
	}
}

func (s *Scheduler) getGroup(name string) *group {
	g, ok := s.groups[name]
	if ok {
		return g
	}

	g = new(group).init(s, name)

	if c, ok := s.cfg.Groups[name]; ok {
		g.Group = *c
	} else {
		g.Note = s.cfg.Group
		g.WorkerNum = s.cfg.WorkerNum
	}

	s.groups[name] = g

	return g
}

func (s *Scheduler) idleFront() *job {
	if s.idle == s.idle.next {
		return nil
	}
	return s.idle.next
}

func (s *Scheduler) idleAdd(j *job) {
	j.mode = job_mode_idle

	jobAppend(j, s.idle.prev)

	s.idleLen++

	//移除多余的idle
	for s.idleLen > int(s.cfg.JobsMaxIdle) {
		j := s.idleFront()
		if j != nil {
			s.idleLen--
			jobRemove(j)
			delete(s.jobs, j.name)
		}
	}
}

func (s *Scheduler) newJob(name string) *job {
	j := new(job).init(s, name)

	c := s.rules.match(name)
	if c != nil {
		j.priority = c.Priority
		j.parallel = c.Parallel
		j.group = c.Group
	} else {
		j.priority = 0
		j.parallel = s.cfg.Parallel
		j.group = s.cfg.Group
	}

	return j
}
