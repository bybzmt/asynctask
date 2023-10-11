package scheduler

import (
	"time"
	"encoding/json"
	"sync"
	"sync/atomic"

	bolt "go.etcd.io/bbolt"
)

type job_mode int

const (
	job_mode_runnable job_mode = iota
	job_mode_block
	job_mode_idle
)

type job struct {
	JobConfig
	TaskBase

	l sync.Mutex

	name string

	s     *Scheduler
	group *group

	next, prev *job
	mode       job_mode

    score int

	nowNum  atomic.Int32
	waitNum atomic.Int32
	errNum  atomic.Int32
	runNum  atomic.Int32
	oldNum  atomic.Int32

	initd bool

	useTimeStat statRow
	loadStat statRow

	loadTime time.Duration
	lastTime time.Time
}

func (j *job) init() error {
	if j.initd {
		return nil
	}
	j.initd = true

	j.useTimeStat.init(10)
	j.loadStat.init(j.s.statSize)

	return j.loadWaitNum()
}

func (j *job) addTask(t *Task) error {
    j.l.Lock()
    defer j.l.Unlock()

	if err := j.init(); err != nil {
		return err
	}

	val, err := json.Marshal(t)
	if err != nil {
		return err
	}

	//key: task/:jname
	err = j.s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "task", j.name)
		if err != nil {
			return err
		}

		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		return bucket.Put([]byte(fmtId(id)), val)
	})

	if err != nil {
		return err
	}

	j.waitNum.Add(1)
	j.s.waitNum.Add(1)

    if j.mode == job_mode_idle {
        j.group.l.Lock()
        defer j.group.l.Unlock()

        j.group.jobs.modeCheck(j)
    }

	return nil
}

func (j *job) delTask(tid ID) error {
	if err := j.init(); err != nil {
		return err
	}

	has := false

	//key: task/:jname
	err := j.s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "task", j.name)
		if err != nil {
			return err
		}

		key := []byte(fmtId(tid))

		v := bucket.Get(key)
		if v != nil {
			has = true
		}

		return bucket.Delete(key)
	})

	if err != nil {
		return err
	}

	if has {
		j.waitNum.Add(-1)
	}

    j.group.l.Lock()
    defer j.group.l.Unlock()

    j.group.jobs.modeCheck(j)

	return nil
}

func (j *job) popTask() (*Task, error) {
	if err := j.init(); err != nil {
		return nil, err
	}

	t := Task{}

	//key: task/:jname
	err := j.s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "task", j.name)
		if err != nil {
			return err
		}

		c := bucket.Cursor()

		for {
			key, val := c.First()

			if key == nil {
				return Empty
			}

			err = bucket.Delete(key)
			if err != nil {
				return err
			}

			err = json.Unmarshal(val, &t)
			if err != nil {
				j.s.Log.Debugln("task", j.name, string(key), "Unmarshal error")

				continue
			}

			return nil
		}
	})

	if err != nil {
		return nil, err
	}

	j.waitNum.Add(-1)
	j.s.waitNum.Add(-1)

	return &t, nil
}

func (j *job) end(now time.Time, useTime time.Duration) {
	j.l.Lock()
	defer j.l.Unlock()

	j.lastTime = now
	j.useTimeStat.push(int64(useTime))
}

func (j *job) hasTask() bool {
	has := false

	//key: task/:jname
	j.s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "task", j.name)
		if bucket == nil {
			return nil
		}

		key, _ := bucket.Cursor().First()

		if key != nil {
			has = true
		}

		return nil
	})

	return has
}

func (j *job) remove() error {
	//key: task/:jname
	err := j.s.Db.Update(func(tx *bolt.Tx) error {

		bucket := getBucket(tx, "task")

		if bucket == nil {
			return nil
		}

		return bucket.DeleteBucket([]byte(j.name))
	})

	return err
}

func (j *job) delAllTask() error {
	err := j.remove()

	if err != nil {
		return err
	}

	j.waitNum.Store(0)

	return nil
}

func (j *job) loadWaitNum() error {
	//key: task/:jname
	err := j.s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "task", j.name)
		if bucket == nil {
			j.waitNum.Store(0)
			return nil
		}

		s := bucket.Stats()

		j.waitNum.Store(int32(s.KeyN))
		j.s.waitNum.Add(int32(s.KeyN))

		return nil
	})

	return err
}

func (j *job) dayChange() {
	n1 := j.runNum.Load()
	j.runNum.Add(-n1)

	n2 := j.errNum.Load()
	j.errNum.Add(-n2)

	j.oldNum.Store(n1)
}


func (j *job) countScore() {
	var x, y, z, area float64

	area = 10000

	x = float64(j.nowNum.Load()) * (area / float64(j.group.WorkerNum))

	if j.group.loadStat.getAll() > 0 {
		y = float64(j.loadStat.getAll()) / float64(j.group.loadStat.getAll()) * area
	}

    xx := j.group.s.waitNum.Load()
	if xx > 0 {
		z = area - float64(j.waitNum.Load())/float64(xx)*area
	}

	j.score = int(x + y + z + float64(j.Priority))
}


func (j *job) popOrder() (*order, error) {
    t, err := j.popTask()
    if err != nil {
        return nil, err
    }

    o := new(order)
    o.Id = ID(t.Id)
    o.Task = t
    o.Base = j.TaskBase
    o.AddTime = time.Unix(int64(t.AddTime), 0)
    o.job = j

    return o, nil
}

