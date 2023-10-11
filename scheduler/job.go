package scheduler

import (
	"time"
	"encoding/json"

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

	name string

	s     *Scheduler
	group *group

	next, prev *job
	mode       job_mode

    score int

	nowNum  int32
	waitNum int32
	errNum  int32
	runNum  int32
	oldNum  int32

	useTimeStat statRow
	loadStat statRow

	loadTime time.Duration
	lastTime time.Time
}

func (j *job) init() error {
	j.useTimeStat.init(10)
	j.loadStat.init(j.s.statSize)

	return j.loadWaitNum()
}

func (j *job) addTask(t *Task) error {
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

	j.waitNum += 1
	j.s.waitNum.Add(1)

    j.group.jobs.modeCheck(j)

	return nil
}

func (j *job) delTask(tid ID) error {
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
		j.waitNum -= 1
	}

    j.group.jobs.modeCheck(j)

	return nil
}

func (j *job) popTask() (*Task, error) {
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

	j.nowNum += 1
	j.waitNum -= 1
	j.s.waitNum.Add(-1)

	return &t, nil
}

func (j *job) end(now time.Time, loadTime, useTime time.Duration) {
	j.nowNum -= 1
	j.runNum += 1
	j.lastTime = now
	j.loadTime += loadTime
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

func (j *job) removeBucket() error {
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
	err := j.removeBucket()

	if err != nil {
		return err
	}

	j.waitNum = 0

	return nil
}

func (j *job) loadWaitNum() error {
	//key: task/:jname
	err := j.s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "task", j.name)
		if bucket == nil {
			j.waitNum = 0
			return nil
		}

		s := bucket.Stats()

		j.waitNum = int32(s.KeyN)
		j.s.waitNum.Add(int32(s.KeyN))

		return nil
	})

	return err
}

func (j *job) dayChange() {
	j.oldNum = j.runNum
	j.runNum = 0
	j.errNum = 0
}


func (j *job) countScore() {
	var x, y, z, area float64

	area = 10000

	x = float64(j.nowNum) * (area / float64(j.group.WorkerNum))

	if j.group.loadStat.getAll() > 0 {
		y = float64(j.loadStat.getAll()) / float64(j.group.loadStat.getAll()) * area
	}

    xx := j.group.s.waitNum.Load()
	if xx > 0 {
		z = area - float64(j.waitNum)/float64(xx)*area
	}

	j.score = int(x + y + z + float64(j.Priority))
}

