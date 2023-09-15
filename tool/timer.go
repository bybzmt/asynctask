package tool

import (
	"encoding/json"
	"fmt"
	"time"

	"asynctask/scheduler"

	bolt "go.etcd.io/bbolt"
)

func TimerRun() {
	tick := time.Tick(time.Second)

	for now := range tick {
		if !hub.Running() {
			return
		}

		timerChecker(now)
	}
}

func timerChecker(now time.Time) {
	trigger := now.Unix()

	// path: /timer/:unix
	err := hub.Db.Update(func(tx *bolt.Tx) error {
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

			var o scheduler.Task
			err = json.Unmarshal(v, &o)
			if err != nil {
				hub.Log.Warnln("[timed] Unmarshal err:", err, "form:", string(v))
			} else {
				err = hub.AddTask(&o)
				if err != nil {
					hub.Log.Warnln("[timed] AddOrder err:", err, "form:", string(v))
				}
			}

			if int64(o.Trigger) > trigger {
				return nil
			}

			err = bucket.Delete(k)
			if err != nil {
				hub.Log.Warnln("[timed] Delete err:", err, "key:", string(k))
				return err
			}
		}
	})

	if err != nil {
		hub.Log.Warnln("[timed] checkTimer err:", err)
	}
}

func timerAddOrder(o *scheduler.Task) error {
	err := hub.Db.Update(func(tx *bolt.Tx) error {
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

		key := fmt.Sprintf("%12d-%12d", o.Trigger, id)

		return bucket.Put([]byte(key), k)
	})

	return err
}

func addTask(o *scheduler.Task) error {
	if o.Trigger > uint(time.Now().Unix()) {
		return timerAddOrder(o)
	}

	return hub.AddTask(o)
}
