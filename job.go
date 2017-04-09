package main

import (
	"container/list"
	"time"
)

type Job struct {
	Name string

	RunNum      int
	CompleteNum int

	Tasks *list.List

	UseTime time.Duration
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

type Jobs struct {
	all     map[string]*list.Element
	alllist *list.List
	use     *list.List
	maxNum  int
}

func (js *Jobs) Init() *Jobs {
	js.all = make(map[string]*list.Element)
	js.use = list.New()
	return js
}

func (js *Jobs) GetJob(action string) *Job {
	return nil
}

func (js *Jobs) HasTask() bool {
	return false
}

func (js *Jobs) GetTask() *Task {
	return nil
}
