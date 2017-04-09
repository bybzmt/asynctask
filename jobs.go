package main

import (
	"github.com/HuKeping/rbtree"
	"log"
	"time"
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
	}

	j.AddTask(t)
}

func (js *Jobs) HasTask() bool {
	return js.use.Len() > 0
}

func (js *Jobs) GetTask() Task {
	j, ok := js.use.Min().(*Job)
	if !ok {
		log.Fatal("gettask", j)
		return Task{}
	}
	if j == nil {
		log.Fatal("gettask", j)
		return Task{}
	}
	log.Println("job", j)

	t := j.PopTask()

	if j.Len() < 1 {
		js.use.Delete(j)
	}

	return t
}
