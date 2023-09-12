package scheduler

import (
	"encoding/json"
	"fmt"
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

    id ID
	Name     string

	g *group

	next, prev *job
	mode       job_mode

	RunNum   int
	OldNum   int
	NowNum   int
	ErrNum   int
	WaitNum  int

    Score int

	LastTime time.Time
	LoadTime time.Duration
	LoadStat StatRow

	//任务执行所用时间
	UseTimeStat StatRow
}

func newJob(js *jobs, name string) *job {
	var cfg JobConfig

	//key: config/job/:name
	err := js.g.s.Db.View(func(tx *bolt.Tx) error {
        bucket := getBucket(tx, "config", "job")
		if bucket == nil {
			return nil
		}

		val := bucket.Get([]byte(name))
		if val == nil {
			return nil
		}

		return json.Unmarshal(val, &cfg)
	})

	if err != nil {
        js.g.s.Log.Warnln("job", name, "config Error", err)
	}

	j := new(job)
	j.JobConfig = cfg
	j.init(name, js.g)

	return j
}

func setJobConfig(db *bolt.DB, name string, cfg JobConfig) error {

	//key: config/job/:name
	err := db.Update(func(tx *bolt.Tx) error {
        bucket := getBucket(tx, "config", "job")
		if bucket == nil {
			return nil
		}

        val, err := json.Marshal(&cfg)
        if err != nil {
            return err
        }

		return bucket.Put([]byte(name), val)
	})

	return err
}

func (j *job) init(name string, g *group) *job {
	j.Name = name
	j.g = g
	j.LoadStat.Init(j.g.s.StatSize)
	j.UseTimeStat.Init(10)
	j.Parallel = j.g.Parallel
	return j
}

func (j *job) addOrder(o *order) error {

	val, err := json.Marshal(o)
	if err != nil {
		return err
	}

    //key: task/:gid/:jname
	err = j.g.s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "task", fmtId(j.g.id), j.Name)
		if err != nil {
			return err
		}

        id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

        key := []byte(fmtId(id))

		return bucket.Put(key, val)
	})

	if err != nil {
		return err
	}

	j.WaitNum++

	return nil
}

func (j *job) delOrder(oid ID) error {
    has:= false

    //key: task/:gid/:jname
    err := j.g.s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "task", fmtId(j.g.id), j.Name)
		if err != nil {
			return err
		}

        key := []byte(fmtId(oid))

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
        j.WaitNum--
    }

    return nil
}

func (j *job) popOrder() (*order, error) {

	o := order{}

	err := j.g.s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "scheduler group", fmtId(j.id))
		if err != nil {
			return err
		}

		c := bucket.Cursor()

		for {
			key, val := c.First()

			if key == nil {
				return nil
			}

			err = bucket.Delete(key)
			if err != nil {
				return err
			}

			err = json.Unmarshal(val, &o)
			if err == nil {
				return nil
			}

			//log
		}
	})

	if err != nil {
		return nil, err
	}

	j.WaitNum--

	return &o, nil
}

func (j *job) delAllTask() error {
    err := j.g.s.Db.Update(func(tx *bolt.Tx) error {
        prefix := []byte("scheduler")
        sid := []byte(fmt.Sprintf("%d", j.g.id))

        bucket, err := tx.CreateBucketIfNotExists(prefix)
        if err != nil {
            return err
        }
        return bucket.DeleteBucket(sid)
    })

	if err != nil {
		return err
	}

    j.WaitNum = 0

    return nil
}

func (j *job) loadLen() error {

	err := j.g.s.Db.View(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "scheduler group", fmtId(j.id))
		if err != nil {
			return err
		}

		s := bucket.Stats()
		j.WaitNum = s.BucketN

		return nil
	})

	return err
}

func (j *job) countScore() {
	var x, y, z, area float64

	area = 10000

	x = float64(j.NowNum) * (area / float64(j.g.WorkerNum))

	if j.g.LoadStat.GetAll() > 0 {
		y = float64(j.LoadStat.GetAll()) / float64(j.g.LoadStat.GetAll()) * area
	}

	if j.g.WaitNum > 0 {
		z = area - float64(j.WaitNum)/float64(j.g.WaitNum)*area
	}

	j.Score = int(x + y + z + float64(j.Priority))
}
