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
	if j.s == nil {
		panic("job Scheduler nil")
	}

	x := 0
	if j.s.LoadStat.GetAll() > 0 {
		x = int(float64(j.LoadStat.GetAll()) / float64(j.s.LoadStat.GetAll()) * 100)
	}

	/*
		y := (200 - j.Len()) / 10
		if y < 0 {
			y = 0
		}
	*/

	return j.NowNum*10 + x
}
