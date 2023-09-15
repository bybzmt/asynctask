package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	bolt "go.etcd.io/bbolt"
)

type Scheduler struct {
	Config

	l sync.Mutex

	now          time.Time
	end          chan *group
	notifyRemove chan string

	taskId ID

	jobTask map[string]*jobTask

	groups  map[ID]*group
	routers []*router

	//统计周期
	statTick time.Duration
	statSize int

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

	s.end = make(chan *group)
	s.groups = make(map[ID]*group)
	s.jobTask = make(map[string]*jobTask)

	s.notifyRemove = make(chan string, 10)

	err := s.loadGroups()
	if err != nil {
		return nil, err
	}

	err = s.loadRouters()
	if err != nil {
		return nil, err
	}

	if len(s.groups) < 1 && len(s.routers) < 1 {
        _, err := s.AddGroup()
        if err != nil {
            return nil, err
        }

        id, err := s.AddRouter()
        if err != nil {
            return nil, err
        }

        r := s.routers[id]
        r.Used = true
        r.Note = "Default Router"
        r.init()
        s.saveRouter(r)
	}

    s.loadJobs()

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
		s.Log.Println("groups")
		go g.Run()
	}

	s.running = true

	s.l.Unlock()

	ticker := time.NewTicker(s.statTick)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			s.Log.Debugln("tick")

			s.l.Lock()
			s.now = now
			for _, s := range s.groups {
				s.tick <- now
			}

			l := len(s.groups)

			s.l.Unlock()

			if !s.running && l == 0 {
				s.Log.Debugln("all close")
				return
			}

		case name := <-s.notifyRemove:
			s.onNotifyRemove(name)
		}
	}
}

func (s *Scheduler) onNotifyRemove(name string) {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobTask[name]
	if ok {
		has := false
		for _, g := range jt.groups {
			g.l.Lock()
			_, ok := g.jobs.all[name];
			g.l.Unlock()

			if ok {
				has = true
                break;
			}
		}

		if !has {
			delete(s.jobTask, name)
		}
	}
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

			if err := g.init(s); err != nil {
				return err
			}

			s.groups[g.Id] = g

			return nil
		})
	})
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
			r.init()

			s.routers = append(s.routers, r)
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
	sort.Slice(s.routers, func(i, j int) bool {
		return s.routers[i].Sort < s.routers[j].Sort
	})
}

func (s *Scheduler) AddGroup() (ID, error) {
	s.l.Lock()
	defer s.l.Unlock()

	g := new(group)
	var cfg GroupConfig

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

		cfg.Id = ID(id)

		val, err := json.Marshal(&cfg)
		if err != nil {
			return err
		}

		if err = bucket.Put([]byte(fmtId(id)), val); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	g.GroupConfig = cfg

	if err := g.init(s); err != nil {
		return 0, err
	}

	s.groups[g.Id] = g

	if s.running {
		go g.Run()
	}

	return g.Id, nil
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

func (s *Scheduler) AddRouter() (ID, error) {
	s.l.Lock()
	defer s.l.Unlock()

	r := new(router)
	var cfg RouteConfig

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
		return 0, err
	}

	r.RouteConfig = cfg
	r.init()

	s.routers = append(s.routers, r)

    s.routersSort()

	return r.Id, nil
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


func (s *Scheduler) routerChanged(r *router) {
    for _, jt := range s.jobTask {
        if r.match(jt.name) {
            var out []*group

            for _, g := range jt.groups {
                has := false
                for _, id := range r.Groups {
                    if id == g.Id {
                        has = true
                        break;
                    }
                }
                if has {
                    out = append(out, g)
                } else {
                    g.notifyDelJob(jt.name)
                }
            }

            for _, id := range r.Groups {
                has := false

                for _, g := range jt.groups {
                    if g.Id == id {
                        has = true
                        break;
                    }
                }

                if !has {
                    if g, ok := s.groups[id]; ok {
                        out = append(out, g)
                    }
                }
            }

            jt.groups = out
        }
    }
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

			_, ok := s.jobTask[name]
			if !ok {
				jt, err := s.addJobTask(name)
				if err != nil {
					return err
				}

				s.jobTask[name] = jt

                for _, g := range jt.groups {
                    g.notifyAddJob(jt)
                }
			}

			return nil
		})
	})

	return err
}

func (s *Scheduler) addJobTask(name string) (*jobTask, error) {
	for _, r := range s.routers {
		if r.match(name) {
			jt := new(jobTask)
			jt.s = s
			jt.base = &r.TaskBase
			jt.name = name

			err := jt.loadWait()
			if err != nil {
				return nil, err
			}

			for _, id := range r.Groups {
				g, ok := s.groups[id]
				if !ok {
					err := errors.New(fmt.Sprintf("router id:%d GroupId:%d not Found", r.Id, id))

					s.Log.Warning(err)

					return nil, err
				}

				jt.groups = append(jt.groups, g)
			}

			return jt, nil
		}
	}

	return nil, errors.New("no match router")
}

func (s *Scheduler) AddTask(t *Task) error {
	s.l.Lock()
	defer s.l.Unlock()

    t.Name = strings.TrimSpace(t.Name)

    if t.Name == "" {
        return TaskError
    }

    if t.Http == nil && t.Cli == nil {
        return TaskError
    }

    s.taskId++
    t.Id = uint(s.taskId)
    t.AddTime = uint(s.now.Unix())

	jt, ok := s.jobTask[t.Name]

	if !ok {
		var err error

		jt, err = s.addJobTask(t.Name)
		if err != nil {
			return err
		}

		s.jobTask[t.Name] = jt
	}

	return jt.addTask(t)
}

func (s *Scheduler) Close() {
	s.l.Lock()

	if !s.running {
		s.l.Unlock()
		return
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

			s.l.Lock()
			l := len(s.groups)
			s.l.Unlock()

			if l == 0 {
				s.Log.Debugln("close end")
				return
			}

		case g := <-s.end:
			s.Log.Debugln("end", g.Id)

			s.l.Lock()
			delete(s.groups, g.Id)
			l := len(s.groups)
			s.l.Unlock()

			if l == 0 {
				close(s.end)
				s.Log.Debugln("Scheduler closd")
				return
			}
		}
	}
}

func (s *Scheduler) allTaskCancel() {
	s.Log.Debugln("allTaskCancel start")
	defer s.Log.Debugln("allTaskCancel end")

	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		g.allTaskCancel()
	}
}
