package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	bolt "go.etcd.io/bbolt"
)

type Scheduler struct {
	Config
	schedulerConfig

	l sync.Mutex

	now          time.Time
	groupEnd     chan *group
	notifyRemove chan string
	closed       chan int

	jobs map[string]*job

	groups map[ID]*group
	routes []*router

	//统计周期
	statTick time.Duration
	statSize int

	waitNum atomic.Int32
	today   int

	running bool
}

func New(c Config) (*Scheduler, error) {
	if c.WorkerNum < 1 {
		c.WorkerNum = 10
	}

	if c.Parallel < 1 {
		c.Parallel = 1
	}

	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	if c.Log == nil {
		c.Log = logrus.StandardLogger()
	}

	s := new(Scheduler)
	s.Config = c
	s.statSize = 60 * 5
	s.statTick = time.Second

	s.groupEnd = make(chan *group)
	s.groups = make(map[ID]*group)
	s.jobs = make(map[string]*job)

	s.notifyRemove = make(chan string, 10)
	s.closed = make(chan int)

	s.loadScheduler()

	if err := s.loadGroups(); err != nil {
		return nil, err
	}

	if err := s.loadRouters(); err != nil {
		return nil, err
	}

	if len(s.groups) < 1 && len(s.routes) < 1 {
		if err := s.AddDefaultGroup(); err != nil {
			return nil, err
		}

		if err := s.AddDefaultRouter(); err != nil {
			return nil, err
		}
	}

	if err := s.loadJobs(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Scheduler) Running() bool {
	s.l.Lock()
	defer s.l.Unlock()

	return s.running
}

func (s *Scheduler) Run() {
	s.l.Lock()

	if s.running {
		panic(errors.New("Run only run once"))
	}

	for _, g := range s.groups {
		go g.Run()
	}

	s.today = time.Now().Day()

	s.running = true

	s.l.Unlock()

	ticker := time.NewTicker(s.statTick)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:

			s.l.Lock()

			s.Log.Debugln("ticker")
			s.now = now

			s.timerChecker(now)

			for _, g := range s.groups {
				g.tick <- now
			}

			l := len(s.groups)

			s.dayCheck()

			s.l.Unlock()

			if !s.running && l == 0 {
				s.Log.Debugln("all close")
				return
			}

		case name := <-s.notifyRemove:
			s.onNotifyRemove(name)

		case g := <-s.groupEnd:
			s.Log.Debugln("groupEnd", g.Id)

			s.l.Lock()
			delete(s.groups, g.Id)
			l := len(s.groups)
			s.l.Unlock()

			if !s.running && l == 0 {
				s.closed <- 1
				return
			}
		}
	}
}

func (s *Scheduler) Close() error {
	s.l.Lock()

	if !s.running {
		s.l.Unlock()
		return nil
	}

	s.running = false

	s.Log.Debugln("Scheduler closing...")

	for _, s := range s.groups {
		s.close()
	}

	s.l.Unlock()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	num := 0

	for {
		select {
		case <-ticker.C:
			num++

			s.Log.Debugln("close tick", num)

			if num == 10 {
				s.allTaskCancel()
			}

		case <-s.closed:
			s.Log.Debugln("Scheduler closd")
			return s.saveScheduler()
		}
	}
}

func (s *Scheduler) allTaskCancel() {
	s.Log.Debugln("allTaskCancel")

	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		g.allTaskCancel()
	}
}

func (s *Scheduler) onNotifyRemove(name string) {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[name]
	if ok {
		if jt.hasTask() {
			return
		}

        jt.group.l.Lock()
        defer jt.group.l.Unlock()

		jt.remove()
		delete(s.jobs, name)
	}
}

func (s *Scheduler) dayCheck() {
	if s.today != s.now.Day() {
		for _, j := range s.jobs {
			j.dayChange()
		}

		for _, g := range s.groups {
			g.dayChange()
		}

		s.today = s.now.Day()
	}
}

func (s *Scheduler) saveScheduler() error {

	//key: config/scheduler.cfg
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config")
		if err != nil {
			return err
		}

		val, err := json.Marshal(&s.schedulerConfig)
		if err != nil {
			return err
		}

		return bucket.Put([]byte("scheduler.cfg"), val)
	})

	return err
}

func (s *Scheduler) loadScheduler() {
	//key: config/scheduler.cfg
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "config")
		if bucket == nil {
			return nil
		}

		var c schedulerConfig

		val := bucket.Get([]byte("scheduler.cfg"))
		if val == nil {
			return nil
		}

		if err := json.Unmarshal(val, &c); err != nil {
			return err
		}

		s.schedulerConfig = c

		return nil
	})

	if err != nil {
		s.Log.Warnln("/config/scheduler.cfg load error:", err)
	}
}

func (s *Scheduler) addGroup() (*group, error) {
	g := new(group)
	var cfg GroupConfig
	cfg.WorkerNum = s.WorkerNum

	//key: config/group/:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config", "group")
		if err != nil {
			return err
		}

		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		val, err := json.Marshal(&cfg)
		if err != nil {
			return err
		}

		if err = bucket.Put([]byte(fmtId(id)), val); err != nil {
			return err
		}

		cfg.Id = ID(id)

		return nil
	})

	if err != nil {
		return nil, err
	}

	g.GroupConfig = cfg
	g.s = s

	s.groups[g.Id] = g

	if s.running {
		go g.Run()
	}

	return g, nil
}

func (s *Scheduler) saveGroup(g *group) error {
	//key: config/group/:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config", "group")
		if err != nil {
			return err
		}

		val, err := json.Marshal(&g.GroupConfig)
		if err != nil {
			return err
		}

		if err = bucket.Put([]byte(fmtId(g.Id)), val); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Scheduler) loadGroups() error {

	//key: config/group/:id
	return s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "config", "group")
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(key, val []byte) error {
			var cfg GroupConfig
			err := json.Unmarshal(val, &cfg)
			if err != nil {
				s.Log.Warnln("[store] key=config/group/"+string(key), "json.Unmarshal:", err)
				return nil
			}

			g := new(group)
			g.GroupConfig = cfg
			g.Id = atoiId(key)
			g.s = s

			s.groups[g.Id] = g

			return nil
		})
	})
}

func (s *Scheduler) addRoute() (*router, error) {
	r := new(router)
	var cfg RouteConfig
	cfg.Parallel = s.Parallel
	cfg.Mode = MODE_HTTP
	cfg.Timeout = 60

	//key: config/router/:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config", "router")
		if err != nil {
			return err
		}

		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		cfg.Id = ID(id)

		key := []byte(fmt.Sprintf("%d", id))
		val, err := json.Marshal(&cfg)

		if err != nil {
			return err
		}

		err = bucket.Put(key, val)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	r.RouteConfig = cfg
	r.init()

	s.routes = append(s.routes, r)

	return r, nil
}

func (s *Scheduler) saveRouter(r *router) error {
	//key: config/router/:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config", "router")
		if err != nil {
			return err
		}

		key := []byte(fmt.Sprintf("%d", r.Id))
		val, err := json.Marshal(&r.RouteConfig)

		if err != nil {
			return err
		}

		err = bucket.Put(key, val)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Scheduler) routeChanged() {
	for _, jt := range s.jobs {
		for _, r := range s.routes {
			if r.match(jt.name) {

				if jt.group != nil && jt.group.Id != r.GroupId {
					jt.group.delJob(jt)
				}

				if g, ok := s.groups[r.GroupId]; ok {
					jt.group = g
				} else {
					jt.group = nil
				}

				jt.TaskBase = r.TaskBase
				jt.JobConfig = r.JobConfig
			}
		}
	}
}

func (s *Scheduler) loadRouters() error {

	//key: config/router/:id
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "config", "router")
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(key, val []byte) error {
			var cfg RouteConfig
			err := json.Unmarshal(val, &cfg)
			if err != nil {
				s.Log.Warnln("[store] key=config/router/"+string(key), "json.Unmarshal:", err)
				return nil
			}

			r := new(router)
			r.RouteConfig = cfg
			r.Id = atoiId(key)
			err = r.init()
			if err != nil {
				return err
			}

			s.routes = append(s.routes, r)
			return nil
		})
	})

	if err != nil {
		return err
	}

	s.routersSort()

	return nil
}

func (s *Scheduler) routersSort() {
	sort.Slice(s.routes, func(i, j int) bool {
		return s.routes[i].Sort > s.routes[j].Sort
	})
}

func (s *Scheduler) addTask(t *Task) error {

	jt, err := s.getJobTask(t.Name)
	if err != nil {
		return err
	}

	return jt.addTask(t)
}

func (s *Scheduler) getJobTask(name string) (*job, error) {
	jt, ok := s.jobs[name]

	if !ok {
		var err error

		jt, err = s.addJobTask(name)
		if err != nil {
			return nil, err
		}

		s.jobs[name] = jt
	}

	return jt, nil
}

func (s *Scheduler) addJobTask(name string) (*job, error) {
	for _, r := range s.routes {
		if r.match(name) {
			jt := new(job)
			jt.s = s
			jt.TaskBase = r.TaskBase
			jt.JobConfig = r.JobConfig
			jt.name = name

			g, ok := s.groups[r.GroupId]
			if !ok {
				err := errors.New(fmt.Sprintf("router id:%d GroupId:%d not Found", r.Id, r.GroupId))

				s.Log.Warning(err)

				return nil, err
			}

			jt.group = g
            jt.init()

            g.addJob(jt)

			return jt, nil
		}
	}

	return nil, errors.New("no match router")
}

func (s *Scheduler) loadJobs() error {
	//key: task/:jname
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "task")
		if bucket == nil {
			return nil
		}

		return bucket.ForEachBucket(func(k []byte) error {
			name := string(k)

			_, ok := s.jobs[name]
			if !ok {
				jt, err := s.addJobTask(name)
				if err != nil {
					return err
				}

				s.jobs[name] = jt
			}

			return nil
		})
	})

	return err
}

func (s *Scheduler) AddDefaultGroup() error {
	s.l.Lock()
	defer s.l.Unlock()

	g, err := s.addGroup()
	if err != nil {
		return err
	}

	g.Note = "Default WorkGroup"

	return s.saveGroup(g)
}

func (s *Scheduler) AddDefaultRouter() error {
	s.l.Lock()
	defer s.l.Unlock()

	r, err := s.addRoute()
	if err != nil {
		return err
	}

	r.Used = true
	r.Note = "Default Router"
	r.Mode = MODE_HTTP

	for _, g := range s.groups {
		r.GroupId = g.Id
	}

	return s.saveRouter(r)
}
