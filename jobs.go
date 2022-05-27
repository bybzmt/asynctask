package main

import (
	"time"
)

type Jobs struct {
	s *Scheduler

	taskId int

	all map[string]*Job

	idleMax int
	idleLen int
	idle    *Job
	block   *Job

	root *Job
}

func (js *Jobs) Init(max int, s *Scheduler) *Jobs {
	js.idleMax = max

	js.all = make(map[string]*Job, max*2)

	js.idle = &Job{}
	js.idle.next = js.idle
	js.idle.prev = js.idle

	js.block = &Job{}
	js.block.next = js.block
	js.block.prev = js.block

	js.root = &Job{}
	js.root.next = js.root
	js.root.prev = js.root

	js.s = s
	return js
}

func (js *Jobs) AddTask(o *Order) {
	j, ok := js.all[o.Name]
	if !ok {
		j = new(Job).Init(o.Name, js.s)
		js.all[o.Name] = j

		//添加到idle链表
		js.idlePushBack(j)
	}

	//从idle移除
	if j.mode == JOB_MODE_IDLE {
		js.idleRmove(j)
		js.pushBack(j)
	}

	j.AddTask(o)

	js.Priority(j)
}

func (js *Jobs) GetTask(now time.Time) *Task {
	j := js.front()
	if j == nil {
		panic("GetTask job is nil")
	}

	t := j.PopTask(now)

	//如果运行数超过上限，移到block队列中
	if j.NowNum >= int(j.parallel) {
		js.remove(j)
		js.blockAdd(j)
	} else if j.Len() < 1 {
		//从运行链表中移聊
		js.remove(j)
		//添加到idle链表中
		js.idlePushBack(j)
	} else {
		js.Priority(j)
	}

	return t
}

func (js *Jobs) end(j *Job, us time.Duration) {
	j.NowNum--
	j.LoadTime += us
	j.UseTimeStat.Push(int64(us))

	if j.mode == JOB_MODE_BLOCK {
		js.remove(j)

		if j.Len() == 0 {
			js.idlePushBack(j)
		} else {
			js.pushBack(j)
		}
	}
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

func (js *Jobs) append(j, at *Job) {
	at.next.prev = j
	j.next = at.next
	j.prev = at
	at.next = j
}

func (js *Jobs) remove(j *Job) {
	j.prev.next = j.next
	j.next.prev = j.prev
	j.next = nil
	j.prev = nil
}

func (js *Jobs) blockAdd(j *Job) {
	j.mode = JOB_MODE_BLOCK
	js.append(j, js.block.prev)
}

func (js *Jobs) pushBack(j *Job) {
	j.mode = JOB_MODE_RUNNABLE
	js.append(j, js.root.prev)
}

func (js *Jobs) idleFront() *Job {
	if js.idle == js.idle.next {
		return nil
	}
	return js.idle.next
}

func (js *Jobs) idlePushBack(j *Job) {
	j.mode = JOB_MODE_IDLE

	js.append(j, js.idle.prev)

	js.idleLen++

	//移除多余的idle
	if js.idleLen > js.idleMax {
		j := js.idleFront()
		if j != nil {
			js.idleRmove(j)
			delete(js.all, j.Name)
		}
	}
}

func (js *Jobs) idleRmove(j *Job) {
	js.remove(j)

	js.idleLen--
}

func (js *Jobs) Priority(j *Job) {
	x := j

	for x.next != nil && x.next != js.root && j.Score() > x.next.Score() {
		x = x.next
	}

	for x.prev != nil && x.prev != js.root && j.Score() < x.prev.Score() {
		x = x.prev
	}

	js.moveBefore(j, x)
}

func (js *Jobs) moveBefore(j, x *Job) {
	if j == x {
		return
	}

	js.remove(j)
	js.append(j, x.prev)
}

func (js *Jobs) Len() int {
	return len(js.all)
}

func (js *Jobs) Each(fn func(j *Job)) {
	for _, j := range js.all {
		fn(j)
	}
}
