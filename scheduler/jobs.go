package scheduler

import (
	"errors"
	"fmt"
	"time"
)

var Empty = errors.New("empty")

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


func (js *jobs) addJob(name string) {
	
	if _, ok := js.all[name]; ok {
        return
    }

    j := newJob(js, name)

    js.all[name] = j

    //添加到idle链表
    js.idlePushBack(j)
}

func (js *jobs) addOrder(o *order) error {
	j, ok := js.all[o.Task.Name]
	if !ok {
		j = newJob(js, o.Task.Name)

		js.all[o.Task.Name] = j

		//添加到idle链表
		js.idlePushBack(j)
	}

	err := j.addOrder(o)
	if err != nil {
		return err
	}

    js.modeCheck(j)

	return nil
}

func (js *jobs) jobEmpty(jid ID) error {
	j := js.getJob(jid)
	if j == nil {
		return errors.New(fmt.Sprintf("job:%d not found", jid))
	}

	err := j.delAllTask()
	if err != nil {
		return err
	}

    js.modeCheck(j)

	return nil
}

func (js *jobs) jobDelIdle(jid ID) error {
	j := js.getJob(jid)
	if j == nil {
		return errors.New(fmt.Sprintf("job:%d not found", jid))
	}

	if j.mode != job_mode_idle {
		return errors.New(fmt.Sprintf("job:%d not idle", jid))
	}

	js.idleRmove(j)
	delete(js.all, j.Name)
	return nil
}

func (js *jobs) jobConfig(name string, cfg JobConfig) error {
    j, ok := js.all[name]
    if !ok {
        return Empty
    }

    j.JobConfig = cfg
    js.modeCheck(j)

	return nil
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
	if j.WaitNum < 1 {
        if j.mode != job_mode_idle {

			js.remove(j)
			js.idlePushBack(j)
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
            js.pushBack(j)
            js.Priority(j)
		}
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
	j.RunNum++
	j.LoadTime += loadTime
	j.UseTimeStat.Push(int64(useTime))

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

func (js *jobs) pushBack(j *job) {
	j.mode = job_mode_runnable
	js.append(j, js.run.prev)
}

func (js *jobs) idleFront() *job {
	if js.idle == js.idle.next {
		return nil
	}
	return js.idle.next
}

func (js *jobs) idlePushBack(j *job) {
	j.mode = job_mode_idle

	js.append(j, js.idle.prev)

	js.idleLen++

	//移除多余的idle
	for js.idleLen > js.idleMax {
		j := js.idleFront()
		if j != nil {
			js.idleRmove(j)
			delete(js.all, j.Name)
		}
	}
}

func (js *jobs) idleRmove(j *job) {
	js.remove(j)

	js.idleLen--
}

func (js *jobs) Priority(j *job) {
	x := j

	for x.next != nil && x.next != js.run && j.Score > x.next.Score {
		x = x.next
	}

	for x.prev != nil && x.prev != js.run && j.Score < x.prev.Score {
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

func (js *jobs) Len() int {
	return len(js.all)
}
