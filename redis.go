package main

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/go-redis/redis"
)

func redis_init() {
	if *redis_host == "" {
		return
	}

	hub.e.Log.Println("[Info] use redis")

	client := redis.NewClient(&redis.Options{
		Addr:     *redis_host,
		Password: *redis_pwd,
		DB:       *redis_db,
	})
	defer client.Close()

	var mem_full = false

	go func() {
		for {
			st := runtime.MemStats{}
			runtime.ReadMemStats(&st)
			if st.Alloc > (*max_mem)*1024*1024 {
				mem_full = true
			} else {
				mem_full = false
			}

			time.Sleep(time.Second * 1)
		}
	}()

	for {
		if !hub.running || mem_full {
			time.Sleep(time.Second * 3)
			continue
		}

		out, err := client.BLPop(time.Second*10, *redis_key).Result()

		if err != nil {
			hub.e.Log.Println("[Debug] redis list empty.", err.Error())
			time.Sleep(time.Second * 3)
		} else {
			o := Order{}
			err = json.Unmarshal([]byte(out[1]), &o)
			if err != nil {
				hub.e.Log.Println("[Debug] redis data Unmarshal error:", err.Error())
			} else {
				ok := hub.AddOrder(&o)
				if !ok {
					out, _ := json.Marshal(&o)
					hub.e.Log.Println("[Info] add Task Fail", out)
				}
			}
		}
	}
}
