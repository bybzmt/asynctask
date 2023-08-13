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

	l sync.Mutex

	now    time.Time
	ticker *time.Ticker
	end    chan *group

	orderId ID

	groups  map[ID]*group
	routers []*router

	//统计周期
	StatTick time.Duration
	StatSize int

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
	s.StatSize = 60 * 5
	s.StatTick = time.Second

	s.groups = make(map[ID]*group)

	err := s.initGroups()
	if err != nil {
		return nil, err
	}

	err = s.initRouters()
	if err != nil {
		return nil, err
	}

	for _, g := range s.groups {
		g.s = s
		err := g.init()
		if err != nil {
			return nil, err
		}
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

	for _, s := range s.groups {
		go s.Run()
	}

	s.running = true
	s.l.Unlock()

	s.ticker = time.NewTicker(s.StatTick)

	for now := range s.ticker.C {
		s.l.Lock()
		s.now = now
		for _, s := range s.groups {
			s.tick <- now
		}
		s.l.Unlock()
	}

	s.ticker.Reset(time.Second * 10)

	go func() {
		for range s.ticker.C {
			s.allTaskCancel()
		}
		s.ticker.Stop()
	}()

	s.l.Lock()
	for _, s := range s.groups {
		s.close()
	}
	s.l.Unlock()

	for g := range s.end {
		s.l.Lock()
		delete(s.groups, g.Id)
		l := len(s.groups)
		s.l.Unlock()

		if l == 0 {
			break
		}
	}

	s.running = false
	close(s.end)

	s.Log.Debugln("Scheduler closd")
}

func (s *Scheduler) initGroups() error {

	//key: config/group/:id
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "config", "group")
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for {
			key, val := c.Next()
			if key == nil {
				break
			}

			var cfg GroupConfig
			err := json.Unmarshal(val, &cfg)
			if err != nil {
				s.Log.Warnln("[store] key=config/group/"+string(key), "json.Unmarshal:", err)
				continue
			}

			g := new(group)
			g.GroupConfig = cfg

			s.groups[g.Id] = g
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(s.groups) < 1 {
		return s.AddGroup()
	}

	return nil
}

func (s *Scheduler) initRouters() error {

	//key: config/router/:id
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "config", "router")
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()

		for {
			key, val := c.Next()
			if key == nil {
				break
			}

			var cfg RouterConfig
			err := json.Unmarshal(val, &cfg)
			if err != nil {
				s.Log.Warnln("[store] key=config/router/"+string(key), "json.Unmarshal:", err)
				continue
			}

			r := new(router)
			r.RouterConfig = cfg

			s.routers = append(s.routers, r)
		}

		return nil
	})

	if err != nil {
		return err
	}

	sort.Slice(s.routers, func(i, j int) bool {
		return s.routers[i].Sort < s.routers[j].Sort
	})

	return nil
}

func (s *Scheduler) AddGroup() error {

	//key: config/group/:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "config", "group")
		if err != nil {
			return err
		}

		var cfg GroupConfig

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

		r := new(group)
		r.GroupConfig = cfg

		s.groups[r.Id] = r

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) AddOrder(t *Task) error {
	s.l.Lock()
	defer s.l.Unlock()

	for _, r := range s.routers {
		if r.match(t) {
			gid := r.randGroup()

			g, ok := s.groups[gid]
			if !ok {
				return errors.New(fmt.Sprintf("group id:%d miss", gid))
			}

			s.orderId++

			o := new(order)
			o.Id = s.orderId
			o.Task = t
			o.AddTime = time.Now()
			o.Base.init()
			copyBase(&o.Base, &g.JobBase)
			copyBase(&o.Base, &r.JobBase)

			return g.addOrder(o)
		}
	}

	return errors.New("no match router")
}

func (s *Scheduler) Close() {
	s.l.Lock()
	defer s.l.Unlock()

	if !s.running {
		return
	}

	s.running = false
	s.Log.Println("Scheduler closing...")

	s.ticker.Stop()
}

func (s *Scheduler) allTaskCancel() {
	s.l.Lock()
	defer s.l.Unlock()

	for _, s := range s.groups {
		s.allTaskCancel()
	}
}
