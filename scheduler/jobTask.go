package scheduler

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	bolt "go.etcd.io/bbolt"
)

type jobTask struct {
	JobConfig
	TaskBase

	l sync.Mutex

	name string

	s     *Scheduler
	group *group

	nowNum  atomic.Int32
	waitNum atomic.Int32
	errNum  atomic.Int32
	runNum  atomic.Int32
	oldNum  atomic.Int32

	initd bool

	useTimeStat statRow

	lastTime time.Time
}

func (j *jobTask) init() error {
	if j.initd {
		return nil
	}
	j.initd = true

	j.useTimeStat.init(10)

	return j.loadWaitNum()
}

func (j *jobTask) addTask(t *Task) error {

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

	if j.group != nil {
		j.group.notifyJob(j)
	}

	return nil
}

func (j *jobTask) delTask(tid ID) error {
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

	if j.group != nil {
		j.group.notifyJob(j)
	}

	return nil
}

func (j *jobTask) popTask() (*Task, error) {
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

func (j *jobTask) end(now time.Time, useTime time.Duration) {
	j.l.Lock()
	defer j.l.Unlock()

	j.lastTime = now
	j.useTimeStat.push(int64(useTime))
}

func (j *jobTask) hasTask() bool {
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

func (j *jobTask) remove() error {
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

func (j *jobTask) delAllTask() error {
	err := j.remove()

	if err != nil {
		return err
	}

	j.waitNum.Store(0)

	if j.group != nil {
		j.group.notifyJob(j)
	}

	return nil
}

func (j *jobTask) loadWaitNum() error {
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

func (j *jobTask) dayChange() {
	n1 := j.runNum.Load()
	j.runNum.Add(-n1)

	n2 := j.errNum.Load()
	j.errNum.Add(-n2)

	j.oldNum.Store(n1)
}
