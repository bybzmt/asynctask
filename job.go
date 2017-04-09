package main

import (
	"container/list"
	"github.com/HuKeping/rbtree"
	"time"
)

type Job struct {
	Name string

	RunNum      int
	CompleteNum int

	Tasks list.List

	UseTime time.Duration
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
	//if e == nil {
	//	panic("PopTask empty")
	//}
	j.Tasks.Remove(e)

	t := e.Value.(Task)
	return t
}

func (j *Job) Len() int {
	return j.Tasks.Len()
}

//红黑树优先级比对
func (j *Job) Less(than rbtree.Item) bool {
	return j.RunNum < than.(*Job).RunNum
}
