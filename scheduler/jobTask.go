package scheduler

import (
	"encoding/json"
	bolt "go.etcd.io/bbolt"
	"sync/atomic"
)

type jobTask struct {
	JobConfig
	TaskBase

	name string

	s      *Scheduler
	groups []*group

	waitNum int32
	errNum  int32
	runNum  int32
	oldNum  int32
}

func (j *jobTask) addTask(t *Task) error {

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

	atomic.AddInt32(&j.waitNum, 1)

	for _, g := range j.groups {
		g.notifyJob(j)
	}

	return nil
}

func (j *jobTask) delTask(tid ID) error {
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
		atomic.AddInt32(&j.waitNum, -1)
	}

	for _, g := range j.groups {
		g.notifyJob(j)
	}

	return nil
}

func (j *jobTask) popTask() (*Task, error) {

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
			if err == nil {
				return nil
			}

			//log
			j.s.Log.Debugln("task/"+j.name, string(key), "Unmarshal error")
		}
	})

	if err != nil {
		return nil, err
	}

	atomic.AddInt32(&j.waitNum, -1)

	return &t, nil
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

	atomic.StoreInt32(&j.waitNum, 0)

	for _, g := range j.groups {
		g.notifyJob(j)
	}

	return nil
}

func (j *jobTask) loadWait() error {
	//key: task/:jname
	err := j.s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "task", j.name)
		if bucket == nil {
			j.waitNum = 0
			return nil
		}

		s := bucket.Stats()
		j.waitNum = int32(s.BucketN)

		return nil
	})

	return err
}


