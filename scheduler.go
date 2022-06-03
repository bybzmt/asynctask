package main

import (
	"container/list"
	"strings"
	"time"
)

type Scheduler struct {
	e *Environment

	//所有工作进程
	allWorkers []*Worker
	//空闲工作进程
	workers *list.List
	//运行中的任务
	tasks map[*Task]struct{}

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
	s.tasks = make(map[*Task]struct{})

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
	if !s.jobs.HasTask() {
		return
	}

	now := time.Now()

	t := s.jobs.GetTask()
	t.StartTime = now
	t.StatTime = now

	s.tasks[t] = struct{}{}

	//得到工人
	ew := s.workers.Front()
	s.workers.Remove(ew)
	w := ew.Value.(*Worker)

	//总状态
	s.NowNum++
	s.RunNum++
	s.WaitNum--

	//分配任务
	t.worker = w
	w.run = true
	w.task <- t
}

func (s *Scheduler) end(t *Task) {
	now := time.Now()

	t.worker.run = false

	t.EndTime = now
	t.worker.log(t)

	us := t.EndTime.Sub(t.StatTime)

	s.jobs.end(t.job, us)

	s.NowNum--
	s.LoadTime += us
	s.workers.PushBack(t.worker)

	delete(s.tasks, t)

	t.job = nil
	t.worker = nil
}

func (s *Scheduler) Run() {
	s.e.Log.Println("[Info] running")

	for i := 1; i <= s.e.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		s.allWorkers = append(s.allWorkers, w)
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
				s.dispatch()
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

	for t, _ := range s.tasks {
		us := now.Sub(t.StatTime)
		t.StatTime = now

		s.LoadTime += us
		t.job.LoadTime += us
	}

	s.LoadStat.Push(int64(s.LoadTime))
	s.LoadTime = 0

	s.jobs.Each(func(j *Job) {
		j.LoadStat.Push(int64(j.LoadTime))
		j.LoadTime = 0
	})

	if s.Today == 0 {
		s.Today = now.Day()
	}

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
