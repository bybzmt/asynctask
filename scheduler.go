package main

import (
	"container/list"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/go-redis/redis"
)

type Cmd int

const (
	CMD_CLOSE Cmd = iota
	CMD_SUSPEND
	CMD_RESUME
)

type Scheduler struct {
	cfg *Config

	//所有工作进程
	allWorkers []*Worker
	//空闲工作进程
	workers *list.List
	//运行中的任务
	tasks map[*Task]struct{}
	//所有任务
	jobs Jobs

	running  bool
	order    chan *Order
	complete chan *Task
	cmd      chan Cmd
	today    int
	memFull  bool

	RunNum  int
	OldNum  int
	NowNum  int
	WaitNum int

	LoadTime time.Duration
	LoadStat StatRow

	redis     *redis.Client
	log       *log.Logger
	logCloser io.Closer
}

func (s *Scheduler) Init(env *Config) *Scheduler {
	s.cfg = env

	s.order = make(chan *Order)
	s.complete = make(chan *Task)
	s.cmd = make(chan Cmd)

	s.jobs.Init(100, s)

	s.workers = list.New()
	s.tasks = make(map[*Task]struct{})

	s.LoadStat.Init(s.cfg.StatSize)

	s.today = time.Now().Day()

	s.openLog()

	return s
}

func (s *Scheduler) addTask(o *Order) {
	s.jobs.AddTask(o)
	s.WaitNum++
}

func (s *Scheduler) dispatch() {
	if !s.running {
		return
	}

	if !s.jobs.HasTask() {
		return
	}

	if s.workers.Len() == 0 {
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
}

func (s *Scheduler) Run() {
	s.log.Println("[Info] running")

	s.running = true

	go s.restoreFromFile()
	go s.redis_init()

	for i := 1; i <= s.cfg.WorkerNum; i++ {
		w := new(Worker).Init(i, s)
		s.allWorkers = append(s.allWorkers, w)
		s.workers.PushBack(w)
		go w.Run()
	}

	tick := time.Tick(s.cfg.StatTick)

	for {
		select {
		case o := <-s.order:
			s.addTask(o)
			s.dispatch()
		case t := <-s.complete:
			s.end(t)
			s.dispatch()
		case now := <-tick:
			s.statTick(now)
			s.dayCheck(now)
			s.memCheck()

			if !s.running {
				if s.workers.Len() == s.cfg.WorkerNum {
					s.log.Println("[Info] all workers closed")
					return
				}
			}
		case c := <-s.cmd:
			switch c {
			case CMD_CLOSE:
				s.close()
			case CMD_SUSPEND:
				for CMD_RESUME != <-s.cmd {
				}
			}
		}
	}
}

func (s *Scheduler) close() {
	if !s.running {
		return
	}

	s.running = false
	s.log.Println("[Info] closing...")

	for _, w := range s.allWorkers {
		w.Close()
	}

	s.saveTask()
}

func (s *Scheduler) statTick(now time.Time) {
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
}

func (s *Scheduler) memCheck() {
	st := runtime.MemStats{}
	runtime.ReadMemStats(&st)
	if st.Alloc > uint64(s.cfg.MaxMem*1024*1024) {
		s.memFull = true
	} else {
		s.memFull = false
	}
}

func (s *Scheduler) dayCheck(now time.Time) {
	if s.today != now.Day() {
		s.OldNum = s.RunNum
		s.RunNum = -1

		s.jobs.Each(func(j *Job) {
			j.OldNum = j.RunNum
			j.RunNum = -1
		})

		s.today = now.Day()
	}
}

func (s *Scheduler) openLog() {
	if s.cfg.LogFile == "" {
		s.log = log.Default()
	} else {
		if s.logCloser != nil {
			s.logCloser.Close()
			s.logCloser = nil
		}

		fh, err := os.OpenFile(s.cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln(err)
		}

		s.log = log.New(fh, "", log.LstdFlags)
		s.logCloser = fh
	}
}
