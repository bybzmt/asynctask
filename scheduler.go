package main

import (
	"container/list"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Order struct {
	Name    string
	Content string
}

type Scheduler struct {
	l sync.Mutex

	WorkerNum int
	BaseUrl   string
	Log       *log.Logger
	Info      *log.Logger
	Client    *http.Client
	timeout   time.Duration

	allWorkers []*Worker
	workers    *list.List

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
	s.WorkerNum = workerNum
	s.BaseUrl = strings.TrimRight(baseurl, "/")

	if out == nil {
		out = log.New(os.Stdout, "[Info] ", log.Ldate)
	}
	if err == nil {
		err = log.New(os.Stderr, "[Scheduler] ", log.LstdFlags)
	}

	s.Info = out
	s.Log = err

	s.order = make(chan Order)
	s.complete = make(chan Task)
	s.cmd = make(chan int)

	s.jobs.Init(1000)

	s.workers = list.New()

	s.timeout = time.Second * 300
	s.stat.StatPeriod = time.Second * 3

	tr := &http.Transport{
		MaxIdleConnsPerHost: s.WorkerNum,
	}

	s.Client = &http.Client{
		Transport: tr,
		Timeout:   s.timeout,
	}

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

func (s *Scheduler) dispatch() {
	t := s.jobs.GetTask()

	ew := s.workers.Front()
	s.workers.Remove(ew)
	w := ew.Value.(*Worker)

	now := time.Now()
	us := now.Sub(w.LastTime)

	w.IdleTime += us
	w.LastTime = now

	s.IdleTime += us
	s.RunNum++

	t.job.RunNum++
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
}

func (s *Scheduler) Run() {
	s.Log.Println("running")

	for i := 1; i <= s.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		w.LastTime = time.Now()
		s.workers.PushBack(w)
		go w.Run()
	}

	go func() {
		t := time.Tick(s.stat.StatPeriod)
		for _ = range t {
			s.cmd <- 2
		}
	}()

	s.running = true

	for {
		select {
		case o := <-s.order:
			s.jobs.AddTask(o)

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
				if s.workers.Len() == s.WorkerNum {
					s.Log.Println("all workers closed")
					return
				}
			}
		case c := <-s.cmd:
			switch c {
			case 1:
				s.running = false

				s.Log.Println("closing...")
				s.saveTask()
			case 2:
				s.stat.LastTime = time.Now()
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
	s.Log.Println("saving tasks...")
}
