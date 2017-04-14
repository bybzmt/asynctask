package main

import (
	"container/list"
	"time"
)

type Job struct {
	s *Scheduler

	next, prev *Job

	Name string

	RunNum int
	NowNum int

	Tasks list.List

	LoadTime time.Duration
	LoadStat StatRow
	LastTime time.Time
}

func (j *Job) Init(name string, s *Scheduler) *Job {
	j.Name = name
	j.Tasks.Init()
	j.s = s
	j.LoadStat.Init(j.s.e.StatSize)
	return j
}

func (j *Job) AddTask(t Task) {
	t.job = j

	j.Tasks.PushBack(t)
}

func (j *Job) PopTask() Task {
	e := j.Tasks.Front()
	if e == nil {
		panic("PopTask empty")
	}
	j.Tasks.Remove(e)

	t, ok := e.Value.(Task)
	if !ok {
		panic("PopTask err")
	}

	return t
}

func (j *Job) Len() int {
	return j.Tasks.Len()
}

func (j *Job) Score() int {
	return j.RunNum
}
