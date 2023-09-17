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
		err := s.Db.Update(func(tx *bolt.Tx) error {
			bucket, err := getBucketMust(tx, "timer")
			if err != nil {
				return err
			}

			for c := bucket.Cursor(); i < 20; {
				k, v := c.Next()
				if k == nil {
					return Empty
				}

				if err = json.Unmarshal(v, &nt[i]); err != nil {
					s.Log.Warnln("[timed] Unmarshal err:", err, "form:", string(v))

					if err := bucket.Delete(k); err != nil {
						return err
					}

					continue
				}

				if int64(nt[i].Trigger) > trigger {
					return Empty
				}

				if err := bucket.Delete(k); err != nil {
					return err
				}

				i++
			}

			return nil
		})

		for x := 0; x < i; {
			if err = s.addTask(&nt[x]); err != nil {
				s.Log.Warnln("[timed] AddOrder err:", err)
			}
		}

		if err != nil {
			if err != Empty {
				s.Log.Warnln("[timed] checkTimer err:", err)
			}

			return
		}
	}
}

func (s *Scheduler) timerAddTask(t *Task) error {

	// path: /timer/:unix-:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "timer")
		if err != nil {
			return err
		}

		k, err := json.Marshal(&t)
		if err != nil {
			return err
		}

		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		key := fmtId(t.Trigger) + "-" + fmtId(id)

		return bucket.Put([]byte(key), k)
	})

	return err
}
