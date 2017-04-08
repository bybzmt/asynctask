package main

import (
	"container/list"
)

type Job struct {
	Name string

	RunNum      int
	CompleteNum int

	Tasks *list.List
}

func (j *Job) Init(name string) *Job {
	j.Name = name
	j.Tasks = list.New()
	return j
}

func (j *Job) AddTask(t Task) {
	t.job = j

	j.Tasks.PushBack(t)
}

func (j *Job) PopTask() Task {
	e := j.Tasks.Front()
	j.Tasks.Remove(e)

	t := e.Value.(Task)
	return t
}

func (j *Job) Len() int {
	return j.Tasks.Len()
}
