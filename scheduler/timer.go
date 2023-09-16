package scheduler

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

func (s *Scheduler) timerChecker(now time.Time) {
	trigger := now.Unix()

    // path: /timer/:unix-:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "timer")
		if err != nil {
			return err
		}

		for {
			c := bucket.Cursor()
			k, v := c.First()
			if k == nil {
				return nil
			}

			var o Task
			err = json.Unmarshal(v, &o)
			if err != nil {
				s.Log.Warnln("[timed] Unmarshal err:", err, "form:", string(v))
			    bucket.Delete(k)
                continue
			}

			if int64(o.Trigger) > trigger {
				return nil
			}

            err = s.addTask(&o)

            if err != nil {
                s.Log.Warnln("[timed] AddOrder err:", err, "form:", string(v))
            }

			err = bucket.Delete(k)
			if err != nil {
				s.Log.Warnln("[timed] Delete err:", err, "key:", string(k))
			}
		}
	})

	if err != nil {
		s.Log.Warnln("[timed] checkTimer err:", err)
	}
}

func (s *Scheduler) timerAddTask(o *Task) error {

    // path: /timer/:unix-:id
	err := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "timer")
		if err != nil {
			return err
		}

		k, err := json.Marshal(&o)
		if err != nil {
			return err
		}

		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

        key := fmtId(o.Trigger) + "-" + fmtId(id)

		return bucket.Put([]byte(key), k)
	})

	return err
}

