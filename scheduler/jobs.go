package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var Empty = errors.New("empty")

type jobs struct {
	g *group

	all map[string]*job

	idleMax int
	idleLen int
	idle    *job
	block   *job

	root *job
}

func (js *jobs) Init(max int, g *group) *jobs {
	js.idleMax = max

	js.all = make(map[string]*job, max*2)

	js.idle = &job{}
	js.idle.next = js.idle
	js.idle.prev = js.idle

	js.block = &job{}
	js.block.next = js.block
	js.block.prev = js.block

	js.root = &job{}
	js.root.next = js.root
	js.root.prev = js.root

	js.g = g
	return js
}

func (js *jobs) addOrder(o *order) error {
	j, ok := js.all[o.Task.Name]
	if !ok {
		j = js.newJob(o)

		js.all[o.Task.Name] = j

		//添加到idle链表
		js.idlePushBack(j)
	}

	err := j.addOrder(o)
	if err != nil {
		return err
	}

    j.countScore()

	//从idle移除
	if j.mode == job_mode_idle {
		js.idleRmove(j)
		js.pushBack(j)
		js.Priority(j)
	}

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

	if j.mode == job_mode_runnable {
		js.remove(j)
		js.idlePushBack(j)
	}

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

func (js *jobs) jobPriority(jid ID, priority int) error {
	j := js.getJob(jid)
	if j == nil {
		return errors.New(fmt.Sprintf("job:%d not found", jid))
	}

    j.Priority = priority

	return nil
}

func (js *jobs) jobParallel(jid ID, parallel uint32) error {
	j := js.getJob(jid)
	if j == nil {
		return errors.New(fmt.Sprintf("job:%d not found", jid))
	}

    j.Parallel = parallel

	return nil
}

func (js *jobs) getJob(jid ID) *job {
	for _, j := range js.all {
		if j.Id == jid {
			return j
		}
	}
	return nil
}

func (js *jobs) newJob(o *order) *job {
	var cfg JobConfig

	//key: config/job/:name
	err := js.g.s.Db.View(func(tx *bolt.Tx) error {
        bucket := getBucket(tx, "config", "job")
		if bucket == nil {
			return nil
		}

		val := bucket.Get([]byte(o.Task.Name))
		if val == nil {
			return nil
		}

		return json.Unmarshal(val, &cfg)
	})

	if err != nil {
        js.g.s.Log.Warnln("job", o.Task.Name, "config Error", err)
	}

	j := new(job)
	j.JobConfig = cfg
	j.Init(o.Task.Name, js.g)

	return j
}

func (js *jobs) modeCheck(j *job) {
	if j.NowNum >= int(j.Parallel) {
		if j.mode == job_mode_runnable {
			js.remove(j)
			js.blockAdd(j)
		}
	} else {
		if j.mode == job_mode_block {
			js.remove(j)

			if j.Len() < 1 {
				js.idlePushBack(j)
			} else {
				js.pushBack(j)
				js.Priority(j)
			}
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

    j.countScore()

	//从运行链表中移聊
	js.remove(j)

	//如果运行数超过上限，移到block队列中
	if j.NowNum >= int(j.Parallel) {
		js.blockAdd(j)
	} else if j.Len() < 1 {
		//添加到idle链表中
		js.idlePushBack(j)
	} else {
		js.pushBack(j)
		js.Priority(j)
	}

	return o, nil
}

func (js *jobs) end(j *job, loadTime, useTime time.Duration) {
	j.NowNum--
	j.RunNum++
	j.LoadTime += loadTime
	j.UseTimeStat.Push(int64(useTime))

    j.countScore()

	if j.mode == job_mode_block {
		if j.NowNum >= int(j.Parallel) {
			return
		}

		js.remove(j)

		if j.Len() < 1 {
			js.idlePushBack(j)
		} else {
			js.pushBack(j)
			js.Priority(j)
		}
	}
}

func (js *jobs) HasTask() bool {
	if js.root == js.root.next {
		return false
	}
	return true
}

func (js *jobs) front() *job {
	if js.root == js.root.next {
		return nil
	}
	return js.root.next
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
	js.append(j, js.root.prev)
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
	if js.idleLen > js.idleMax {
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

	for x.next != nil && x.next != js.root && j.Score > x.next.Score {
		x = x.next
	}

	for x.prev != nil && x.prev != js.root && j.Score < x.prev.Score {
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
