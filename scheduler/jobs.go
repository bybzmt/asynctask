package scheduler

import (
	"time"
)

type jobs struct {
	g *group

	all map[string]*job

	idleMax int
	idleLen int
	idle    *job
	block   *job

	run *job
}

func (js *jobs) init(max int, g *group) *jobs {
	js.idleMax = max

	js.all = make(map[string]*job, max*2)

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


func (js *jobs) addJob(jtask *jobTask) *job {
	
	if j, ok := js.all[jtask.name]; ok {
        return j
    }

    j := newJob(js, jtask)

    js.all[jtask.name] = j

    //添加到idle链表
    js.idleAdd(j)

    return j
}

func (js *jobs) getJob(jid ID) *job {
	for _, j := range js.all {
		if j.id == jid {
			return j
		}
	}
	return nil
}

func (js *jobs) modeCheck(j *job) {
    if j.next == nil || j.prev == nil {
        return
    }

	if !j.hasTask() {
        if j.mode != job_mode_idle {
			js.remove(j)
			js.idleAdd(j)
        }
    } else if j.NowNum >= int(j.Parallel) {
		if j.mode != job_mode_block {
			js.remove(j)
            js.blockAdd(j)
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

	o, err := j.popOrder()
	if err != nil {
		return nil, err
	}

    o.job = j

	j.LastTime = js.g.now
	j.NowNum++

    js.modeCheck(j)

	return o, nil
}

func (js *jobs) end(j *job, loadTime, useTime time.Duration) {
	j.NowNum--
	j.runAdd()
	j.LoadTime += loadTime
	j.UseTimeStat.push(int64(useTime))

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
			js.idleRmove(j)
			delete(js.all, j.Name)

            js.g.s.notifyRemove <- j.Name

            j.task = nil
		}
	}
}

func (js *jobs) idleRmove(j *job) {
	js.remove(j)

	js.idleLen--
}

func (js *jobs) priority(j *job) {
	x := j

	for x.next != js.run && j.Score > x.next.Score {
		x = x.next
	}

	for x.prev != js.run && j.Score < x.prev.Score {
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

func (js *jobs) len() int {
	return len(js.all)
}
