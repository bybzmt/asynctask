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
	parallel   int

	Name string

	RunNum int
	OldNum int
	NowNum int

	Tasks list.List

	LoadTime time.Duration
	LoadStat StatRow
	LastTime time.Time

	UseTimeStat StatRow
}

func (j *Job) Init(name string, s *Scheduler) *Job {
	j.Name = name
	j.Tasks.Init()
	j.s = s
	j.LoadStat.Init(j.s.e.StatSize)
	j.UseTimeStat.Init(10)
	j.parallel = j.s.e.Parallel
	return j
}

func (j *Job) AddTask(o *Order) {

	j.s.jobs.taskId++

	t := &Task{
		job:     j,
		Id:      o.Id,
		Params:  o.Params,
		AddTime: time.Unix(int64(o.AddTime), 0),
	}

	parallel := o.Parallel
	if parallel < 1 || parallel > j.s.e.Parallel {
		parallel = j.s.e.Parallel
	}

	j.parallel = parallel

	j.Tasks.PushBack(t)
}

func (j *Job) PopTask(now time.Time) *Task {
	e := j.Tasks.Front()
	if e == nil {
		panic("PopTask empty")
	}
	j.Tasks.Remove(e)

	t, ok := e.Value.(*Task)
	if !ok {
		panic("PopTask err")
	}

	//任务状态
	j.LastTime = now
	j.NowNum++
	j.RunNum++

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

	x := j.NowNum * (area / j.s.e.WorkerNum)

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
