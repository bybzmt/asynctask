package main

import (
	"container/list"
	"github.com/HuKeping/rbtree"
	"time"
)

type Jobs struct {
	all    Lru
	use    *rbtree.Rbtree
	use2   list.List
	taskId int
}

func (js *Jobs) Init(max int) *Jobs {
	js.all.Init(max)
	js.use = rbtree.New()
	js.use2.Init()
	return js
}

func (js *Jobs) AddTask(o Order) {
	var j *Job

	ji, ok := js.all.Get(o.Name)
	if ok {
		j = ji.(*Job)
	} else {
		j = new(Job).Init(o.Name)
		js.all.Add(o.Name, j)
	}

	js.taskId++

	t := Task{
		job:     j,
		Id:      js.taskId,
		Content: o.Content,
		AddTime: time.Now(),
	}

	if j.Len() == 0 {
		js.use.Insert(j)
		//js.use2.PushBack(j)
	}

	j.AddTask(t)
}

func (js *Jobs) HasTask() bool {
	return js.use.Len() > 0
	//return js.use2.Len() > 0
}

func (js *Jobs) GetTask() Task {
	j, ok := js.use.Min().(*Job)

	//je := js.use2.Front()
	//j, ok := je.Value.(*Job)
	if !ok || j == nil {
		panic("GetTask job is nil")
	}

	t := j.PopTask()

	if j.Len() < 1 {
		js.use.Delete(j)
		//js.use2.Remove(je)
	}

	return t
}
