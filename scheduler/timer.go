package scheduler

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

func (s *Scheduler) timerChecker(now time.Time) {
	trigger := now.Unix()

	for {
		i := 0
		nt := make([]Task, 20)

		// path: /timer/:unix-:id
		s.Db.Update(func(tx *bolt.Tx) error {
			bucket, err := getBucketMust(tx, "timer")
			if err != nil {
				s.Log.Error("[timed] getBucketMust err:", err)
				return nil
			}

			c := bucket.Cursor()

			for i < 20 {
				k, v := c.First()
				if k == nil {
					return nil
				}

				if err = json.Unmarshal(v, &nt[i]); err != nil {
					s.Log.Error("[timed] Unmarshal err:", err, "json:", string(v))
				} else {
					if int64(nt[i].Trigger) > trigger {
						return nil
					} else {
						i++
					}
				}

				if err := bucket.Delete(k); err != nil {
					s.Log.Error("[timed] Delete err:", err, "key:", string(k))
					return nil
				}
			}

			return nil
		})

		for x := 0; x < i; x++ {
			if err := s.addTask(&nt[x]); err != nil {
				s.Log.Warnln("[timed] AddOrder err:", err)
			}
		}

		if i == 0 {
			return
		}
	}
}

func (s *Scheduler) timerAddTask(t *Task) error {

	if _, err := s.getJob(t.Name); err != nil {
		return err
	}

	// path: /timer/:unix-:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "timer")
		if err != nil {
			return err
		}

		val, err := json.Marshal(&t)
		if err != nil {
			return err
		}

		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		key := fmtId(t.Trigger) + "-" + fmtId(id)

		return bucket.Put([]byte(key), val)
	})

	return err
}

func (s *Scheduler) timerTaskNum() int {
    num := 0

	// path: /timer
	s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "timer")
		if bucket == nil {
			return nil
		}

		stat := bucket.Stats()

        num = stat.KeyN

		return nil
	})

    return num
}
