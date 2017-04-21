package main

import (
	"container/list"
	"log"
	"strings"
	"time"
)

type Order struct {
	Name    string `json:"action"`
	Content string `json:"params"`
}

type Scheduler struct {
	e *Environment

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

func (s *Scheduler) Init(workerNum int, baseurl string, out, err *log.Logger) *Scheduler {
	s.e = new(Environment).Init(workerNum, baseurl, out, err)
	s.e.StatTick = time.Second * 1
	s.e.StatSize = 60

	s.order = make(chan *Order)
	s.complete = make(chan *Task)
	s.cmd = make(chan int)
	s.statResp = make(chan *Statistics)

	s.jobs.Init(2000, s)

	s.workers = list.New()

	s.LoadStat.Init(s.e.StatSize)
	s.IdleStat.Init(s.e.StatSize)

	return s
}

func (s *Scheduler) AddOrder(name, content string) bool {
	if !s.running {
		return false
	}

	name = strings.Trim(name, "/ ")
	if name == "" {
		return false
	}

	o := &Order{
		Name:    name,
		Content: content,
	}

	s.order <- o

	return true
}

func (s *Scheduler) addTask(o *Order) {
	j := s.jobs.getJob(o.Name)

	s.WaitNum++
	s.jobs.taskId++

	t := &Task{
		job:     j,
		Id:      s.jobs.taskId,
		Content: o.Content,
		AddTime: time.Now(),
	}

	j.AddTask(t)

	if j.Len() == 1 {
		s.jobs.PushBack(j)
	}

	s.jobs.Priority(j)
}

func (s *Scheduler) dispatch() {
	j := s.jobs.GetTaskJob()

	t := j.PopTask()

	now := time.Now()

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

	//任务状态
	j.LastTime = now
	j.NowNum++
	j.RunNum++

	if j.Len() < 1 {
		s.jobs.Remove(j)
	} else {
		s.jobs.Priority(j)
	}

	//分配任务
	t.worker = w
	w.run = true
	w.task <- t
}

func (s *Scheduler) end(t *Task) {
	now := time.Now()
	t.worker.LastTime = now
	t.worker.run = false

	us := t.EndTime.Sub(t.StartTime)

	t.job.NowNum--
	t.job.LoadTime += us
	t.job.UseTimeStat.Push(int64(us))

	s.NowNum--
	s.LoadTime += us
	s.workers.PushBack(t.worker)
	s.jobs.Priority(t.job)
}

func (s *Scheduler) Run() {
	s.e.Log.Println("running")

	for i := 1; i <= s.e.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		s.e.allWorkers = append(s.e.allWorkers, w)
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
					s.e.Log.Println("all workers closed")
					return
				}
			}
		case c := <-s.cmd:
			switch c {
			case 1:
				//关闭
				s.running = false

				s.e.Log.Println("closing...")
				s.saveTask()
			case 2:
				if !s.running {
					if s.workers.Len() == s.e.WorkerNum {
						s.e.Log.Println("all workers closed")
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

	for _, w := range s.e.allWorkers {
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

func (s *Scheduler) getStatData() {
	e1 := s.LoadStat.GetAll() + s.IdleStat.GetAll()
	all := 0
	if e1 > 0 {
		all = int(float64(s.LoadStat.GetAll()) / float64(s.LoadStat.GetAll()+s.IdleStat.GetAll()) * 10000)
	}

	t := &Statistics{}
	t.Jobs = make([]Stat, 0, s.jobs.Len())
	t.All.Name = "all"
	t.All.Load = all
	t.All.RunNum = s.RunNum
	t.All.OldNum = s.OldNum
	t.All.NowNum = s.NowNum
	t.All.WaitNum = s.WaitNum

	s.jobs.Each(func(j *Job) {
		x := 0
		if e1 > 0 {
			x = int(float64(j.LoadStat.GetAll()) / float64(e1) * 10000)
		}

		t.Jobs = append(t.Jobs, Stat{
			Name:    j.Name,
			Load:    x,
			RunNum:  j.RunNum,
			OldNum:  j.OldNum,
			NowNum:  j.NowNum,
			WaitNum: j.Len(),
			UseTime: int(j.UseTimeStat.GetAll()/int64(time.Millisecond)) / len(j.UseTimeStat.data),
		})
	})

	s.statResp <- t
}

func (s *Scheduler) Close() {
	s.cmd <- 1
}

func (s *Scheduler) WaitClosed() {
}

func (s *Scheduler) saveTask() {
	s.e.Log.Println("saving tasks...")
	s.saveToFile()
	s.e.Log.Println("saving tasks complete")
}

func (s *Scheduler) Status() *Statistics {
	s.cmd <- 3
	t := <-s.statResp
	return t
}
