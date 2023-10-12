package scheduler

import (
	"time"
)

type jobs struct {
	g *group

	idleMax int
	idleLen int

	idle  *job
	block *job
	run   *job
}

func (js *jobs) init(max int, g *group) *jobs {
	js.idleMax = max

	js.idle = &job{}
	js.idle.next = js.idle
	js.idle.prev = js.idle

	js.block = &job{}
	js.block.next = js.block
	js.block.prev = js.block

	js.run = &job{}
	js.run.next = js.run
	js.run.prev = js.run

	js.g = g
	return js
}

func (js *jobs) addJob(j *job) {
	js.runAdd(j)

	js.g.jobs.modeCheck(j)
}

func (js *jobs) modeCheck(j *job) {
	if j.next == nil || j.prev == nil {
		js.g.s.Log.Warning("modeCheck nil")
		return
	}

	if j.nowNum >= int32(j.Parallel) || (j.waitNum < 1 && j.nowNum > 0) {
		if j.mode != job_mode_block {
			js.remove(j)
			js.blockAdd(j)
		}
	} else if j.waitNum < 1 {
		if j.mode != job_mode_idle {
			js.remove(j)
			js.idleAdd(j)
		}
	} else {
		j.countScore()

		if j.mode != job_mode_runnable {
			js.remove(j)
			js.runAdd(j)
		}

		js.priority(j)
	}
}

func (js *jobs) GetOrder() (*order, error) {
	j := js.front()
	if j == nil {
		return nil, Empty
	}

	o, err := js.popOrder(j)

	if err != nil {
		if err == Empty {
            js.g.s.Log.Warnln("Job PopOrder Empty")

            j.waitNum = 0
			js.modeCheck(j)
		}
		return nil, err
	}

	js.modeCheck(j)

	return o, nil
}

func (js *jobs) popOrder(j *job) (*order, error) {
	t, err := j.popTask()
	if err != nil {
		return nil, err
	}

	o := new(order)
	o.Id = ID(t.Id)
	o.Task = t
	o.Base = copyTaskBase(j.TaskBase)
	o.AddTime = time.Unix(int64(t.AddTime), 0)
	o.job = j

	return o, nil
}

func (js *jobs) end(j *job, loadTime, useTime time.Duration) {
	j.end(js.g.now, loadTime, useTime)

	js.modeCheck(j)
}

func (js *jobs) front() *job {
	if js.run == js.run.next {
		return nil
	}
	return js.run.next
}

func (js *jobs) append(j, at *job) {
	at.next.prev = j
	j.next = at.next
	j.prev = at
	at.next = j
}

func (js *jobs) remove(j *job) {
	j.prev.next = j.next
	j.next.prev = j.prev
	j.next = nil
	j.prev = nil
}

func (js *jobs) blockAdd(j *job) {
	j.mode = job_mode_block
	js.append(j, js.block.prev)
}

func (js *jobs) runAdd(j *job) {
	j.mode = job_mode_runnable
	js.append(j, js.run.prev)
}

func (js *jobs) idleFront() *job {
	if js.idle == js.idle.next {
		return nil
	}
	return js.idle.next
}

func (js *jobs) idleAdd(j *job) {
	j.mode = job_mode_idle

	js.append(j, js.idle.prev)

	js.idleLen++

	//移除多余的idle
	for js.idleLen > js.idleMax {
		j := js.idleFront()
		if j != nil {
            js.removeJob(j)
		}
	}
}

func (js *jobs) removeJob(j *job) bool {
	if j.mode != job_mode_idle {
		return false
	}

	js.remove(j)
	js.idleLen--

	return true
}

func (js *jobs) priority(j *job) {
	x := j

	for x.next != js.run && j.score > x.next.score {
		x = x.next
	}

	for x.prev != js.run && j.score < x.prev.score {
		x = x.prev
	}

	js.moveBefore(j, x)
}

func (js *jobs) moveBefore(j, x *job) {
	if j == x {
		return
	}

	js.remove(j)
	js.append(j, x.prev)
}
