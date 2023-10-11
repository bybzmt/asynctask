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
	//添加到idle链表
	js.idleAdd(j)

	js.g.jobs.modeCheck(j)
}

func (js *jobs) modeCheck(j *job) {
	if j.next == nil || j.prev == nil {
		js.g.s.Log.Warning("modeCheck nil")
		return
	}

	nowNum := j.nowNum.Load()

	if nowNum >= int32(j.Parallel) {
		if j.mode != job_mode_block {
			js.remove(j)
			js.blockAdd(j)
		}
	} else if !j.hasTask() {
		if nowNum > 0 {
			if j.mode != job_mode_block {
				js.remove(j)
				js.blockAdd(j)
			}
		} else {
			if j.mode != job_mode_idle {
				js.remove(j)
				js.idleAdd(j)
			}
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
		if js.block.next != js.block {
			js.modeCheck(js.block.next)
		} else {
			// for x := js.idle; x.next != js.idle; x = x.next {
			//              if x.hasTask() {
			//                  js.modeCheck(x)
			//                  break;
			//              }
			// }
		}

		return nil, Empty
	}

	o, err := j.popOrder()

	if err != nil {
		if err == Empty {
			js.modeCheck(j)
		}
		return nil, err
	}

	j.nowNum.Add(1)

	js.modeCheck(j)

	return o, nil
}

func (js *jobs) end(j *job, loadTime, useTime time.Duration) {
	j.nowNum.Add(-1)
	j.runNum.Add(1)
	j.loadTime += loadTime

	j.end(js.g.now, useTime)

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
			js.g.s.notifyRemove <- j.name
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

	for x.next != js.run && x.score > x.next.score {
		x = x.next
	}

	for x.prev != js.run && x.score < x.prev.score {
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
