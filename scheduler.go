package main

import (
	"container/list"
	"log"
	"strings"
	"time"
)

type Order struct {
	Name    string
	Content string
}

type Scheduler struct {
	e *Environment

	workers *list.List

	jobs Jobs

	running  bool
	order    chan Order
	complete chan Task
	cmd      chan int

	RunNum   int
	UseTime  time.Duration
	IdleTime time.Duration

	stat Statistics
}

func (s *Scheduler) Init(workerNum int, baseurl string, out, err *log.Logger) *Scheduler {
	s.e = new(Environment).Init(workerNum, baseurl, out, err)

	s.order = make(chan Order)
	s.complete = make(chan Task)
	s.cmd = make(chan int)

	s.jobs.Init(1000)

	s.workers = list.New()

	return s
}

func (s *Scheduler) AddOrder(name, content string) bool {
	if !s.running {
		return false
	}

	o := Order{
		Name:    strings.TrimLeft(name, "/"),
		Content: content,
	}

	s.order <- o

	return true
}

func (s *Scheduler) addTask(o Order) {
	j := s.jobs.getJob(o.Name)

	s.jobs.taskId++

	t := Task{
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
	j := s.jobs.getTaskJob()

	t := j.PopTask()

	now := time.Now()

	//得到工人
	ew := s.workers.Front()
	s.workers.Remove(ew)
	w := ew.Value.(*Worker)

	//工人空闲间
	us := now.Sub(w.LastTime)
	w.IdleTime += us
	w.LastTime = now

	s.IdleTime += us
	s.RunNum++

	j.LastTime = now
	j.RunNum++

	if j.Len() < 1 {
		s.jobs.Remove(j)
	} else {
		s.jobs.Priority(j)
	}

	t.worker = w
	w.task <- t
}

func (s *Scheduler) end(t Task) {
	now := time.Now()
	us := now.Sub(t.worker.LastTime)

	t.worker.TaskNum++
	t.worker.UseTime += us
	t.worker.LastTime = now

	t.job.RunNum--
	t.job.CompleteNum++
	t.job.UseTime += us

	s.UseTime += us
	s.workers.PushBack(t.worker)

	s.jobs.Priority(t.job)
}

func (s *Scheduler) Run() {
	s.e.Log.Println("running")

	for i := 1; i <= s.e.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		w.LastTime = time.Now()
		s.workers.PushBack(w)
		go w.Run()
	}

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
				s.running = false

				s.e.Log.Println("closing...")
				s.saveTask()
			case 2:
			}
		}
	}
}

func (s *Scheduler) Close() {
	s.cmd <- 1
}

func (s *Scheduler) WaitClosed() {
}

func (s *Scheduler) saveTask() {
	s.e.Log.Println("saving tasks...")
}
