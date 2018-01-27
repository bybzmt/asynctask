package main

import (
	"container/list"
	"time"
)

type Job struct {
	s *Scheduler

	next, prev *Job

	Method string
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

func (j *Job) Init(method, name string, s *Scheduler) *Job {
	j.Method = method
	j.Name = name
	j.Tasks.Init()
	j.s = s
	j.LoadStat.Init(j.s.e.StatSize)
	j.UseTimeStat.Init(10)
	return j
}

func (j *Job) AddTask(t *Task) {
	t.job = j

	j.Tasks.PushBack(t)
}

func (j *Job) PopTask() *Task {
	e := j.Tasks.Front()
	if e == nil {
		panic("PopTask empty")
	}
	j.Tasks.Remove(e)

	t, ok := e.Value.(*Task)
	if !ok {
		panic("PopTask err")
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
