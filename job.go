package main

import (
	"container/list"
	"time"
)

type job_mode int

const (
	JOB_MODE_RUNNABLE job_mode = iota
	JOB_MODE_BLOCK
	JOB_MODE_IDLE
)

type Job struct {
	s *Scheduler

	next, prev *Job
	mode       job_mode
	parallel   uint

	Name string

	RunNum int
	OldNum int
	NowNum int

	Tasks list.List

	LoadTime time.Duration
	LoadStat StatRow

	//任务执行所用时间
	UseTimeStat StatRow
}

func (j *Job) Init(name string, s *Scheduler) *Job {
	j.Name = name
	j.Tasks.Init()
	j.s = s
	j.LoadStat.Init(j.s.cfg.StatSize)
	j.UseTimeStat.Init(10)
	j.parallel = j.s.cfg.Parallel
	return j
}

func (j *Job) AddTask(o *Order) {

	j.s.jobs.taskId++

	t := &taskMini{
		Id:      o.Id,
		Params:  o.Params,
		AddTime: o.AddTime,
	}

	if o.Parallel > 0 {
		j.parallel = o.Parallel
	}

	j.Tasks.PushBack(t)
}

func (j *Job) PopTask() *Task {
	e := j.Tasks.Front()
	if e == nil {
		panic("PopTask empty")
	}
	j.Tasks.Remove(e)

	m, ok := e.Value.(*taskMini)
	if !ok {
		panic("PopTask err")
	}

	t := &Task{
		job:     j,
		Id:      m.Id,
		Params:  m.Params,
		AddTime: time.Unix(int64(m.AddTime), 0),
	}

	return t
}

func (j *Job) Len() int {
	return j.Tasks.Len()
}

func (j *Job) Score() int {
	if j.s == nil {
		panic("job Scheduler nil")
	}

	area := 10000

	x := j.NowNum * (area / j.s.cfg.WorkerNum)

	y := 0
	if j.s.LoadStat.GetAll() > 0 {
		y = int(float64(j.LoadStat.GetAll()) / float64(j.s.LoadStat.GetAll()) * float64(area))
	}

	z := 0
	if j.s.WaitNum > 0 {
		z = area - int(float64(j.Len())/float64(j.s.WaitNum)*float64(area))
	}

	return x*4 + y*3 + z*3
}

func (j *Job) Each(fn func(t *taskMini)) {
	if j.Len() > 0 {
		ele := j.Tasks.Front()
		for ele != nil {
			t := ele.Value.(*taskMini)

			fn(t)

			ele = ele.Next()
		}
	}
}
