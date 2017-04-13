package main

import ()

type Jobs struct {
	all    Lru
	taskId int

	front, back *Job
	size        int
}

func (js *Jobs) Init(max int) *Jobs {
	js.all.Init(max)
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
	return js.size > 0
}

func (js *Jobs) getTaskJob() *Job {
	if js.front == nil {
		panic("GetTask job is nil")
	}

	return js.front
}

func (js *Jobs) PushBack(j *Job) {
	js.size++

	if js.back == nil {
		js.back = j
		js.front = j
		j.next = nil
		j.prev = nil
		return
	}

	js.back.next = j
	j.prev = js.back
	j.next = nil
	js.back = j
}

func (js *Jobs) Priority(j *Job) {
}

func (js *Jobs) Remove(j *Job) {
	if j.prev != nil {
		j.prev.next = j.next
	}

	if j.next != nil {
		j.next.prev = j.prev
	}

	if j == js.front {
		js.front = j.next
	}

	if j == js.back {
		js.back = j.prev
	}

	j.prev = nil
	j.next = nil

	js.size--
}
