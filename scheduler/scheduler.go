package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	bolt "go.etcd.io/bbolt"
)

type Scheduler struct {
	Config
	schedulerConfig

	l sync.Mutex

	now    time.Time
	closed chan int

	idleMax int
	idleLen int

	idle *job

	jobs map[string]*job

	groups map[ID]*group
	routes []*router

	orders   map[*order]struct{}
	complete chan *order

	//统计周期
	statTick time.Duration
	statSize int

	today    int
	timedNum int

	running bool
}

func (s *Scheduler) Init() error {
	if s.WorkerNum < 1 {
		s.WorkerNum = 10
	}

	if s.Parallel < 1 {
		s.Parallel = 1
	}

	if s.Client == nil {
		s.Client = http.DefaultClient
	}

	if s.Log == nil {
		l := logrus.StandardLogger()

		l.SetLevel(logrus.InfoLevel)
		l.SetFormatter(&logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})

		s.Log = l
	}

	s.statSize = 60 * 5
	s.statTick = time.Second

	s.groups = make(map[ID]*group)
	s.jobs = make(map[string]*job)
	s.complete = make(chan *order)
	s.orders = make(map[*order]struct{})
	s.closed = make(chan int)

	s.idleMax = 200
	s.idle = &job{}
	s.idle.next = s.idle
	s.idle.prev = s.idle

	if err := s.init(); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) Running() bool {
	s.l.Lock()
	defer s.l.Unlock()

	return s.running
}

func (s *Scheduler) init() error {
	s.loadScheduler()

	if err := s.loadGroups(); err != nil {
		return err
	}

	if err := s.loadRouters(); err != nil {
		return err
	}

	if len(s.groups) < 1 && len(s.routes) < 1 {
		if err := s.addDefaultGroup(); err != nil {
			return err
		}

		if err := s.addDefaultRouter(); err != nil {
			return err
		}
	}

	if err := s.loadJobs(); err != nil {
		return err
	}

	s.timedNum = s.timerTaskNum()

	return nil
}

func (s *Scheduler) Run() {
    s.Log.Debugln("run start")
	defer s.Log.Debugln("run stop")

	s.l.Lock()

	if s.running {
		panic(errors.New("Run only run once"))
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

			s.dayCheck()

			s.statMaintain(now)

			l := len(s.orders)

			s.timerChecker(now)

			if s.running {
				for _, g := range s.groups {
					g.dispatch()
				}
			}

			s.l.Unlock()

			if !s.running && l == 0 {
				s.closed <- 1
				return
			}

		case o := <-s.complete:
			s.l.Lock()

			o.g.end(o)

			if s.running {
                s.now = time.Now()

				for o.g.dispatch() {
				}
			} else {
				if len(s.orders) == 0 {
					s.closed <- 1
                    return
				}
			}

			s.l.Unlock()
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

func (s *Scheduler) checkJobs() {
	for _, j := range s.jobs {
		if j.mode == job_mode_idle {
			j.loadWaitNum()
			if j.waitNum > 0 {
				j.group.jobs.addJob(j)
			}

			if j.next == nil {
				delete(s.jobs, j.name)
			}
		}
	}
}

func (s *Scheduler) allTaskCancel() {
	s.Log.Debugln("allTaskCancel")

	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		g.cancel()
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
	g.init()

	s.groups[g.Id] = g

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
			g.init()

			s.groups[g.Id] = g

			return nil
		})
	})
}

func (s *Scheduler) addRoute() (*router, error) {
	r := new(router)
	var cfg TaskConfig
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

	r.TaskConfig = cfg
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
		val, err := json.Marshal(&r.TaskConfig)

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
	for _, j := range s.jobs {
		for _, r := range s.routes {
			if r.match(j.name) {
				if j.group.Id != r.GroupId {
                    jobRemove(j)

					g, ok := s.groups[r.GroupId]

					if !ok {
						delete(s.jobs, j.name)
						s.Log.Errorf("routeChanged Miss Group route:%d job:%s\n", r.Id, j.name)
						continue
					}

                    n := new(job)
                    *n = *j

					n.group = g
					g.jobs.addJob(n)

                    s.jobs[n.name] = n
                    j = n
				}

                j.JobConfig = r.JobConfig
                j.TaskBase = copyTaskBase(r.TaskBase)
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
			var cfg TaskConfig
			err := json.Unmarshal(val, &cfg)
			if err != nil {
				s.Log.Warnln("[store] key=config/router/"+string(key), "json.Unmarshal:", err)
				return nil
			}

			r := new(router)
			r.TaskConfig = cfg
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

	j, err := s.getJob(t.Name)
	if err != nil {
		return err
	}

	return j.addTask(t)
}

func (s *Scheduler) getJob(name string) (*job, error) {
	jt, ok := s.jobs[name]

	if !ok {
		var err error

		jt, err = s.addJob(name)
		if err != nil {
			return nil, err
		}

		s.jobs[name] = jt
	}

	return jt, nil
}

func (s *Scheduler) addJob(name string) (*job, error) {
	for _, r := range s.routes {
		if r.match(name) {
			j := new(job)
			j.s = s
			j.TaskBase = r.TaskBase
			j.JobConfig = r.JobConfig
			j.name = name

			g, ok := s.groups[r.GroupId]
			if !ok {
				err := errors.New(fmt.Sprintf("router id:%d GroupId:%d not Found", r.Id, r.GroupId))

				s.Log.Warning(err)

				return nil, err
			}

			j.group = g
			j.init()

			return j, nil
		}
	}

	return nil, errors.New("Task Not Allow")
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
				jt, err := s.addJob(name)
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

func (s *Scheduler) addDefaultGroup() error {
	s.l.Lock()
	defer s.l.Unlock()

	g, err := s.addGroup()
	if err != nil {
		return err
	}

	g.Note = "Default"

	return s.saveGroup(g)
}

func (s *Scheduler) addDefaultRouter() error {
	s.l.Lock()
	defer s.l.Unlock()

	r, err := s.addRoute()
	if err != nil {
		return err
	}

	r.Used = true
	r.Note = "Default"
	r.Mode = MODE_HTTP

	for _, g := range s.groups {
		r.GroupId = g.Id
	}

	return s.saveRouter(r)
}
