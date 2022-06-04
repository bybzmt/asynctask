package main

import (
	"container/list"
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
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
	allWorkers []Worker
	//空闲工作进程
	workers list.List
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

	//己执行任务计数
	RunNum int
	//昨天任务计数
	OldNum int
	//执行中的任务
	NowNum int
	//总队列长度
	WaitNum int

	//负载数据
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

	s.workers.Init()
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

func (s *Scheduler) dispatch(now time.Time) {
	if !s.running {
		return
	}

	if !s.jobs.HasTask() {
		return
	}

	if s.workers.Len() == 0 {
		return
	}

	t := s.jobs.GetTask()
	t.StartTime = now
	t.StatTime = now

	s.tasks[t] = struct{}{}

	//得到工人
	ew := s.workers.Front()
	s.workers.Remove(ew)
	w := ew.Value.(Worker)

	//总状态
	s.NowNum++
	s.RunNum++
	s.WaitNum--

	//分配任务
	t.worker = w

	w.Exec(t)
}

func (s *Scheduler) end(t *Task, now time.Time) {
	t.EndTime = now

	s.logTask(t)

	loadTime := t.EndTime.Sub(t.StatTime)
	useTime := t.EndTime.Sub(t.StartTime)

	s.jobs.end(t.job, loadTime, useTime)

	s.NowNum--
	s.LoadTime += loadTime

	s.workers.PushBack(t.worker)

	delete(s.tasks, t)
}

func (s *Scheduler) Run() {
	s.log.Println("[Info] running")

	s.running = true

	go s.restoreFromFile()
	go s.redis_init()

	for i := 1; i <= s.cfg.WorkerNum; i++ {
		var w Worker

		if s.cfg.Mode == MODE_CMD {
			w = new(WorkerCmd).Init(i, s)
		} else {
			w = new(WorkerHttp).Init(i, s)
		}

		s.allWorkers = append(s.allWorkers, w)
		s.workers.PushBack(w)
		go w.Run()
	}

	tick := time.Tick(s.cfg.StatTick)

	for {
		select {
		case o := <-s.order:
			s.addTask(o)
			now := time.Now()
			s.dispatch(now)
		case t := <-s.complete:
			now := time.Now()
			s.end(t, now)
			s.dispatch(now)
		case now := <-tick:
			s.statTick(now)
			s.dayCheck(now)
			s.memCheck()

			if !s.running {
				if s.workers.Len() == s.cfg.WorkerNum {
					s.closed()
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

func (s *Scheduler) closed() {
	s.log.Println("[Info] all workers closed")
	if s.logCloser != nil {
		s.logCloser.Close()
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

		file := s.cfg.LogFile
		file = strings.Replace(file, "[date]", time.Now().Format("20060102"), 1)

		fh, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln(err)
		}

		s.log = log.New(fh, "", log.LstdFlags)
		s.logCloser = fh
	}
}

func (s *Scheduler) logTask(t *Task) {

	var waitTime float64 = 0
	if t.AddTime.Unix() > 0 {
		waitTime = t.StartTime.Sub(t.AddTime).Seconds()
	}

	runTime := t.EndTime.Sub(t.StartTime).Seconds()

	d := TaskLog{
		Id:       t.Id,
		Name:     t.job.Name,
		Params:   t.Params,
		Status:   t.Status,
		WaitTime: LogSecond(waitTime),
		RunTime:  LogSecond(runTime),
		Output:   t.Msg,
	}

	msg, _ := json.Marshal(d)

	s.log.Printf("[Task] %s\n", msg)
}
