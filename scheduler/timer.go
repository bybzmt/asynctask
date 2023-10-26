package scheduler

import (
	"encoding/json"
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

func (s *Scheduler) timerChecker(now time.Time) {
	trigger := now.Unix()

	var brk = errors.New("brk")

	page := 20

	for {
		nt := make([]Task, 0, page)

		empty := true

		// path: /timer/:unix-:id
		s.Db.Update(func(tx *bolt.Tx) error {
			bucket, err := getBucketMust(tx, "timer")
			if err != nil {
				s.Log.Error("[timed] getBucketMust err:", err)
				return nil
			}

			keys := make([][]byte, 0, page)

			bucket.ForEach(func(k, v []byte) error {
				empty = false

				t := Task{}

				if err = json.Unmarshal(v, &t); err != nil {
					s.Log.WithFields(map[string]any{
						"tag":  "timed",
						"task": string(v),
					}).Errorln("Unmarshal err:", err)

					keys = append(keys, k)
					return nil
				}

				if int64(t.Timer) > trigger {
					return brk
				}

				nt = append(nt, t)
				keys = append(keys, k)

				if len(nt) >= page {
					return brk
				}

				return nil
			})

			for _, k := range keys {
				if err := bucket.Delete(k); err != nil {
					s.Log.WithFields(map[string]any{
						"tag": "timed",
						"key": string(k),
					}).Errorln("Delete err:", err)
				}
			}

			return nil
		})

		if empty {
			s.timedNum = 0
		}

		if len(nt) == 0 {
			return
		}

		s.timedNum -= len(nt)

		for _, t := range nt {
			if err := s.addTask(&t); err != nil {
				v, _ := json.Marshal(t)
				s.Log.WithFields(map[string]any{
					"tag":  "timed",
					"task": string(v),
				}).Infoln("AddOrder err:", err)
			}
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

		key := fmtId(t.Timer) + "-" + fmtId(id)

		return bucket.Put([]byte(key), val)
	})

	if err == nil {
		s.timedNum++
	}

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

type timedTask struct {
	Task
	TimedID string
}

func (s *Scheduler) TimerShow(starttime, num int) (out []timedTask) {

	// path: /timer/:unix-:id
	s.Db.View(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, "timer")
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		k, v := c.Seek([]byte(fmtId(starttime)))

		for i := 0; k != nil && i < num; i++ {
			t := timedTask{}

			if err := json.Unmarshal(v, &t); err == nil {
				t.TimedID = string(k)

				out = append(out, t)
			}

			k, v = c.Next()
		}

		return nil
	})

	return
}

func (s *Scheduler) TimerDel(TimedID string) error {
	// path: /timer/:unix-:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "timer")
		if err != nil {
			return err
		}

		return bucket.Delete([]byte(TimedID))
	})

	return err
}
