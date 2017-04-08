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

	orders map[string]*Job

	jobs    *list.List
	workers *list.List

	taskId int

	running  bool
	order    chan Order
	complete chan Task
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

	s.orders = make(map[string]*Job)
	s.order = make(chan Order)
	s.complete = make(chan Task)
	s.jobs = list.New()
	s.workers = list.New()

	s.timeout = time.Second * 300

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

func (s *Scheduler) add(o Order) {
	j, ok := s.orders[o.Name]
	if !ok {
		j = new(Job).Init(o.Name)
		s.orders[o.Name] = j
	}

	s.taskId++

	t := Task{
		job:     j,
		Id:      s.taskId,
		Content: o.Content,
		AddTime: time.Now(),
	}

	if j.Len() == 0 {
		s.jobs.PushBack(j)
	}

	j.AddTask(t)
}

func (s *Scheduler) dispatch() {
	et := s.jobs.Front()
	j := et.Value.(*Job)

	t := j.PopTask()
	if j.Len() == 0 {
		s.jobs.Remove(et)
	} else {
		s.jobs.MoveToBack(et)
	}

	j.RunNum++

	ew := s.workers.Front()
	s.workers.Remove(ew)

	w := ew.Value.(*Worker)
	w.task <- t
}

func (s *Scheduler) end(t Task) {
	s.workers.PushBack(t.worker)

	t.worker.TaskNum++
	t.job.RunNum--
	t.job.CompleteNum++

	t.job = nil
	t.worker = nil
}

func (s *Scheduler) Run() {
	s.Log.Println("running")

	for i := 1; i <= s.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		go w.Run()
		s.workers.PushBack(w)
	}

	s.running = true

	for {
		select {
		case o := <-s.order:
			s.add(o)

			if s.workers.Len() > 0 {
				s.dispatch()
			}
		case t := <-s.complete:
			if t.Id == -1 {
				s.Log.Println("closing...")
				s.saveTask()

				s.running = false
			} else {
				s.end(t)

				if s.running {
					if s.jobs.Len() > 0 {
						s.dispatch()
					}
				} else {
					if s.workers.Len() == s.WorkerNum {
						s.Log.Println("all workers closed")
						return
					}
				}
			}
		}
	}
}

func (s *Scheduler) Close() {
	s.complete <- Task{Id: -1}
}

func (s *Scheduler) WaitClosed() {
}

func (s *Scheduler) saveTask() {
	s.Log.Println("saving tasks...")
}
