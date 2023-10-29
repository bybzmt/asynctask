package scheduler

import (
	"encoding/json"
	"errors"
	"net/http"
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
	router router
	rules  []Rule

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

func (s *Scheduler) init() error {
	s.loadScheduler()

	var err error
	s.router.routes, err = s.db_router_load()
	if err != nil {
		return err
	}

	s.router.init()

	if err := s.loadGroups(); err != nil {
		return err
	}

	if err := s.rulesReload(); err != nil {
		return err
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

func (s *Scheduler) allTaskCancel() {
	s.Log.Debugln("allTaskCancel")

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

	return j.removeBucket()
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

func (s *Scheduler) addTask(t *order) error {
	j, ok := s.jobs[t.Job]

	if !ok {
		j = s.newJob(t.Job)
		s.jobs[t.Job] = j
	}

	return j.addTask(t)
}

func (s *Scheduler) getGroup(id ID) *group {
	g, ok := s.groups[id]
	if ok {
		return g
	}

	s.Log.Warnf("group id:%d not found\n", id)

	g, ok = s.groups[1]
	if ok {
		return g
	}

	g = new(group)
	g.Id = 1
	g.s = s
	g.Note = "Default"
    g.WorkerNum = s.WorkerNum
	g.init()

	s.db_group_save(g)

	s.groups[1] = g

	return g
}
