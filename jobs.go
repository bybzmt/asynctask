package main

import (
	"github.com/HuKeping/rbtree"
)

type Jobs struct {
	all    Lru
	use    *rbtree.Rbtree
	taskId int
}

func (js *Jobs) Init(max int) *Jobs {
	js.all.Init(max)
	js.use = rbtree.New()
	return js
}

func (js *Jobs) getJob(name string) *Job {
	var j *Job

	ji, ok := js.all.Get(name)
	if ok {
		j, ok = ji.(*Job)
		if !ok {
			panic("getJob err")
		}
	} else {
		j = new(Job).Init(name)
		js.all.Add(name, j)
	}

	return j
}

func (js *Jobs) HasTask() bool {
	return js.use.Len() > 0
}

func (js *Jobs) getTaskJob() *Job {
	je := js.use.Min()
	if je == nil {
		panic("GetTask job is nil")
	}

	j, ok := je.(*Job)
	if !ok || j == nil {
		panic("GetTask job is nil")
	}

	return j
}
