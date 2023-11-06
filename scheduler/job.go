package scheduler

import (
	"time"

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
	name string

	cfgMode int

	s     *Scheduler
	group *group

	next, prev *job
	mode       job_mode

	score int
    empty bool

	nowNum  int32
	waitNum int32
	errNum  int32
	runNum  int32
	oldRun  int32
	oldErr  int32

	useTimeStat statRow
	loadStat    statRow

	loadTime time.Duration
	lastTime time.Time
}

func (j *job) init() error {
	j.useTimeStat.init(10)
	j.loadStat.init(j.s.statSize)

	return j.loadWaitNum()
}

func (j *job) addTask(t *Order) error {
	//key: task/:jname
	err := db_push(j.s.Db, t, "task", j.name)
	if err != nil {
		return err
	}

    j.empty = false

	j.waitNum += 1
	j.group.waitNum += 1

	if j.next == nil || j.prev == nil {
		j.group.runAdd(j)
	}

	j.group.modeCheck(j)

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
		j.group.waitNum -= 1
	}

	return nil
}

func (j *job) popTask() (*Order, error) {
	t := new(Order)

	err := db_shift(j.s.Db, &t, "task", j.name)
	if err != nil {
		if err == Empty {
            j.empty = true
			j.waitNum = 0
		}

		return nil, err
	}

	j.nowNum++
	j.waitNum--
	j.group.waitNum -= 1

	return t, nil
}

func (j *job) end(now time.Time, loadTime, useTime time.Duration) {
	j.nowNum--
	j.runNum++
	j.lastTime = now
	j.loadTime += loadTime
	j.useTimeStat.push(int64(useTime))
}

func (j *job) removeBucket() error {
	//key: task/:jname
	return db_bucket_del(j.s.Db, "task", j.name)
}

func (j *job) delAllTask() error {
	err := j.removeBucket()

	if err != nil {
		return err
	}

	j.group.waitNum -= int(j.waitNum)
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
		j.group.waitNum += s.KeyN

		return nil
	})

	return err
}

func (j *job) dayChange() {
	j.oldRun = j.runNum
	j.oldErr = j.errNum
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

	xx := j.group.waitNum
	if xx > 0 {
		z = area - float64(j.waitNum)/float64(xx)*area
	}

	j.score = int(x + y + z + float64(j.Priority))
}
