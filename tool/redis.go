package tool

import (
	"encoding/json"
	"strconv"
	"time"

	"asynctask/scheduler"
	"github.com/go-redis/redis"
)

func RedisRun(addr, pwd, db, key string) {
	hub.Log.Println("[Info] redis init")
	defer hub.Log.Println("[Info] redis close")

	_db, _ := strconv.Atoi(db)

	redis := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       _db,
	})

	for {
		out, err := redis.BLPop(time.Second*3, key).Result()

		if err != nil {
			hub.Log.Debugln("redis list empty.", err.Error())

			if !hub.Running() {
				return
			}

			time.Sleep(time.Second)
		} else {
			data := out[1]

			o := scheduler.Task{}
			err = json.Unmarshal([]byte(data), &o)
			if err != nil {
				hub.Log.Warnln("redis data Unmarshal error:", err.Error(), data)
			} else {
				err := addOrder(&o)
				if err != nil {
					hub.Log.Warnln("redis add Task Fail", data)
				}
			}
		}
	}
}
