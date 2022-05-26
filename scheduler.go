package main

import (
	"container/list"
	"strings"
	"time"
)

type Order struct {
	Id       uint32   `json:"id,omitempty"`
	Parallel int      `json:"parallel,omitempty"`
	Name     string   `json:"name"`
	Params   []string `json:"params,omitempty"`
	AddTime  int64    `json:"add_time,omitempty"`
}

type Scheduler struct {
	e *Environment

	//所有工作进程
	allWorkers []*Worker
	//空闲工作进程
	workers *list.List

	jobs Jobs

	running  bool
	order    chan *Order
	complete chan *Task
	cmd      chan int

	statResp chan *Statistics

	RunNum int
	OldNum int
	Today  int

	NowNum  int
	WaitNum int

	LoadTime time.Duration
	IdleTime time.Duration

	LoadStat StatRow
	IdleStat StatRow
}

func (s *Scheduler) Init(env *Environment) *Scheduler {
	s.e = env
	s.e.StatTick = time.Second * 1
	s.e.StatSize = 30

	s.order = make(chan *Order)
	s.complete = make(chan *Task)
	s.cmd = make(chan int)
	s.statResp = make(chan *Statistics)

	s.jobs.Init(100, s)

	s.workers = list.New()

	s.LoadStat.Init(s.e.StatSize)
	s.IdleStat.Init(s.e.StatSize)

	return s
}

func (s *Scheduler) AddOrder(o *Order) bool {
	if !s.running {
		return false
	}

	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return false
	}

	s.order <- o

	return true
}

func (s *Scheduler) addTask(o *Order) {
	s.jobs.AddTask(o)
	s.WaitNum++
}

func (s *Scheduler) dispatch() {
	now := time.Now()

	t := s.jobs.GetTask(now)

	//得到工人
	ew := s.workers.Front()
	s.workers.Remove(ew)
	w := ew.Value.(*Worker)

	//空闲间
	us := now.Sub(w.LastTime)

	//总状态
	s.NowNum++
	s.RunNum++
	s.WaitNum--
	s.IdleTime += us

	//分配任务
	t.worker = w
	w.run = true
	w.task <- t
}

func (s *Scheduler) end(t *Task) {
	t.worker.LastTime = t.EndTime
	t.worker.run = false

	us := t.EndTime.Sub(t.StartTime)

	s.jobs.end(t.job, us)

	s.NowNum--
	s.LoadTime += us
	s.workers.PushBack(t.worker)
}

func (s *Scheduler) Run() {
	s.e.Log.Println("[Info] running")

	for i := 1; i <= s.e.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		s.allWorkers = append(s.allWorkers, w)
		w.LastTime = time.Now()
		s.workers.PushBack(w)
		go w.Run()
	}

	go func() {
		c := time.Tick(s.e.StatTick)
		for _ = range c {
			s.cmd <- 2
		}
	}()

	s.running = true

	for {
		select {
		case o := <-s.order:
			s.addTask(o)

			if s.workers.Len() > 0 {
				s.dispatch()
			}
		case t := <-s.complete:
			s.end(t)

			if s.running {
				if s.jobs.HasTask() {
					s.dispatch()
				}
			} else {
				if s.workers.Len() == s.e.WorkerNum {
					s.e.Log.Println("[Info] all workers closed")
					return
				}
			}
		case c := <-s.cmd:
			switch c {
			case 1:
				//关闭
				s.running = false

				s.e.Log.Println("[Info] closing...")
				s.saveTask()
			case 2:
				if !s.running {
					if s.workers.Len() == s.e.WorkerNum {
						s.e.Log.Println("[Info] all workers closed")
						return
					}
				}
				s.statTick()
			case 3:
				s.getStatData()
			}
		}
	}
}

func (s *Scheduler) statTick() {
	//时间片统计
	now := time.Now()

	if s.Today == 0 {
		s.Today = now.Day()
	}

	for _, w := range s.allWorkers {
		if !w.run {
			us := now.Sub(w.LastTime)
			s.IdleTime += us
			w.LastTime = now
		}
	}

	s.LoadStat.Push(int64(s.LoadTime))
	s.IdleStat.Push(int64(s.IdleTime))
	s.LoadTime = 0
	s.IdleTime = 0

	s.jobs.Each(func(j *Job) {
		j.LoadStat.Push(int64(j.LoadTime))
		j.LoadTime = 0
	})

	if s.Today != now.Day() {
		s.OldNum = s.RunNum
		s.RunNum = 0

		s.jobs.Each(func(j *Job) {
			j.OldNum = j.RunNum
			j.RunNum = 0
		})

		s.Today = now.Day()
	}
}

func (s *Scheduler) Close() {
	s.cmd <- 1
}
