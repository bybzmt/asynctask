package scheduler

import (
	"time"
)

type jobs struct {
	block *job
	run   *job
}

func (js *jobs) init() {
	js.block = &job{}
	js.block.next = js.block
	js.block.prev = js.block

	js.run = &job{}
	js.run.next = js.run
	js.run.prev = js.run
}

func (js *jobs) addJob(j *job) {
	js.runAdd(j)

	js.modeCheck(j)
}

func (js *jobs) modeCheck(j *job) {
	if j.next == nil || j.prev == nil {
		j.s.Log.Warning("modeCheck nil")
		return
	}

	if j.nowNum >= int32(j.Parallel) || (j.waitNum < 1 && j.nowNum > 0) {
		if j.mode != job_mode_block {
			jobRemove(j)
			js.blockAdd(j)
		}
	} else if j.waitNum < 1 {
		if j.mode != job_mode_idle {
			jobRemove(j)
			j.s.idleAdd(j)
		}
	} else {
		j.countScore()

		if j.mode != job_mode_runnable {
			jobRemove(j)
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
            j.s.Log.Warnln("Job PopOrder Empty")

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
    o.g = j.group
	o.Task = t
	o.Base = copyTaskBase(j.TaskBase)
	o.AddTime = time.Unix(int64(t.AddTime), 0)
	o.job = j

	return o, nil
}

func (js *jobs) end(j *job, loadTime, useTime time.Duration) {
	j.end(j.s.now, loadTime, useTime)

	js.modeCheck(j)
}

func (js *jobs) front() *job {
	if js.run == js.run.next {
		return nil
	}
	return js.run.next
}

func (js *jobs) blockAdd(j *job) {
	j.mode = job_mode_block
	jobAppend(j, js.block.prev)
}

func (js *jobs) runAdd(j *job) {
	j.mode = job_mode_runnable
	jobAppend(j, js.run.prev)
}

func (js *jobs) priority(j *job) {
	x := j

	for x.next != js.run && j.score > x.next.score {
		x = x.next
	}

	for x.prev != js.run && j.score < x.prev.score {
		x = x.prev
	}

    jobMoveBefore(j, x)
}

func (s *Scheduler) idleFront() *job {
	if s.idle == s.idle.next {
		return nil
	}
	return s.idle.next
}

func (s *Scheduler) idleAdd(j *job) {
	j.mode = job_mode_idle

	jobAppend(j, s.idle.prev)

	s.idleLen++

	//移除多余的idle
	for s.idleLen > s.idleMax {
		j := s.idleFront()
		if j != nil {
			jobRemove(j)
			s.idleLen--
		}
	}
}


