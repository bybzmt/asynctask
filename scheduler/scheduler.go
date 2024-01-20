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

	//统计周期
	statTick time.Duration
	statSize int

	today int

	running bool

	ctx    context.Context
	cancel context.CancelFunc
}

func (s *Scheduler) init() error {

	if s.log == nil {
		s.log = new(nullLogger)
	}

	s.running = true
	s.today = time.Now().Day()

	s.statSize = 60 * 5
	s.statTick = time.Second

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

func (s *Scheduler) SetConfig(c *Config) error {
	s.l.Lock()
	defer s.l.Unlock()

	if c.Dirver == nil {
		return DirverError
	}

	groups := make(map[string]*group)

	for k := range c.Groups {
		if g, ok := s.groups[k]; ok {
			groups[k] = g
		}
	}

	for _, j := range c.Jobs {
		_, ok := groups[j.Group]
		if !ok {
			return errors.New("Job:" + j.Pattern + " Group:" + j.Group + " Not Found")
		}
	}

	var j rules

	if err := j.set(c.Jobs); err != nil {
		return err
	}

	s.cfg = *c
	s.groups = groups
	s.rules = j

	//default value

	if s.cfg.Group == "" {
		s.cfg.Group = "default"
	}

	if s.cfg.WorkerNum == 0 {
		s.cfg.WorkerNum = 10
	}

	if s.cfg.Parallel == 0 {
		s.cfg.Parallel = 1
	}

	if s.cfg.Timeout == 0 {
		s.cfg.Timeout = 60
	}

	if s.cfg.JobsMaxIdle == 0 {
		s.cfg.JobsMaxIdle = 100
	}

	if s.cfg.CloseWait == 0 {
		s.cfg.CloseWait = 10
	}

	s.log = s.cfg.Log
	if s.log == nil {
		s.log = new(nullLogger)
	}

	//apply config
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

	s.dirver = c.Dirver

	return nil
}

func (s *Scheduler) Run() {
	s.log.Println("run start")
	defer s.log.Println("run stop")

	if s.running {
		panic(errors.New("Run only run once"))
	}

	ticker := time.NewTicker(s.statTick)
	defer ticker.Stop()

	var num uint

	for {
		select {
		case now := <-ticker.C:
			s.onTick(now)

			if !s.running {
				num++

				s.log.Println("close tick", num)

				if num == s.cfg.CloseWait {
					s.log.Println("allTaskCancel")
					s.cancel()
				}
			}

		case o := <-s.complete:
			s.onComplete(o)
		}

		if !s.running {
			s.l.Lock()
			l := len(s.orders)
			s.l.Unlock()

			if l == 0 {
				s.log.Println("Scheduler closd")
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

	for _, g := range s.groups {
		for g.dispatch() {
		}
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

func (s *Scheduler) Close() {
	s.l.Lock()
	defer s.l.Unlock()

	if !s.running {
		return
	}

	s.log.Println("Scheduler closing...")

	s.running = false

	for _, g := range s.groups {
		g.WorkerNum = 0
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
