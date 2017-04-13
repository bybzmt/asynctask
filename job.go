package main

import (
	"container/list"
	"time"
)

type Job struct {
	next, prev *Job

	Name string

	RunNum      int
	CompleteNum int

	Tasks list.List

	UseTime  time.Duration
	LastTime time.Time
}

func (j *Job) Init(name string) *Job {
	j.Name = name
	j.Tasks.Init()
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
