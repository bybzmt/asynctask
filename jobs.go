package main

import ()

type Jobs struct {
	s *Scheduler

	all    Lru
	taskId int

	root *Job
}

func (js *Jobs) Init(max int, s *Scheduler) *Jobs {
	js.all.Init(max)
	js.root = &Job{}
	js.root.next = js.root
	js.root.prev = js.root
	js.s = s
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
		j = new(Job).Init(name, js.s)
		js.all.Add(name, j)
	}

	return j
}

func (js *Jobs) HasTask() bool {
	if js.root == js.root.next {
		return false
	}
	return true
}

func (js *Jobs) front() *Job {
	if js.root == js.root.next {
		return nil
	}
	return js.root.next
}

func (js *Jobs) GetTaskJob() *Job {
	j := js.front()
	if j == nil {
		panic("GetTask job is nil")
	}

	return j
}

func (js *Jobs) append(j, at *Job) {
	at.next.prev = j
	j.next = at.next
	j.prev = at
	at.next = j
}

func (js *Jobs) PushBack(j *Job) {
	js.append(j, js.root.prev)
}

func (js *Jobs) Priority(j *Job) {
	x := j

	for x.next != nil && x.next != js.root && j.Score() > x.next.Score() {
		x = x.next
	}

	for x.prev != nil && x.prev != js.root && j.Score() < x.prev.Score() {
		x = x.prev
	}

	js.MoveBefore(j, x)
}

func (js *Jobs) MoveBefore(j, x *Job) {
	if j == x {
		return
	}

	js.Remove(j)
	js.append(j, x.prev)
}

func (js *Jobs) Remove(j *Job) {
	j.prev.next = j.next
	j.next.prev = j.prev
	j.next = nil
	j.prev = nil
}

func (js *Jobs) Len() int {
	return js.all.Len()
}

func (js *Jobs) Each(fn func(j *Job)) {
	js.all.Each(func(k, v interface{}) {
		j, ok := v.(*Job)
		if !ok {
			panic("Jobs Each Data Type err")
		}

		fn(j)
	})
}
