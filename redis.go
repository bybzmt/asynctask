package main

import (
	"encoding/json"
	"flag"
	"github.com/go-redis/redis"
	"runtime"
	"time"
)

var redis_host = flag.String("redis_host", "", "redis host")
var redis_pwd = flag.String("redis_pwd", "", "redis password")
var redis_db = flag.Int("redis_db", 0, "redis database")
var redis_key = flag.String("redis_key", "", "redis list key name. json data: {action:string, params:string}")
var max_mem = flag.Uint64("max_mem", 128, "max memory size(MB)")

func redis_init() {
	if *redis_host == "" {
		return
	}

	hub.e.Log.Println("use redis")

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
			time.Sleep(time.Second * 1)
			continue
		}

		out, err := client.BLPop(time.Second*5, *redis_key).Result()
		if err != nil {
			hub.e.Info.Println("redis list empty.", err.Error())
		} else {
			o := Order{}
			err = json.Unmarshal([]byte(out[1]), &o)
			if err != nil {
				hub.e.Log.Println("redis data Unmarshal error:", err.Error())
			}

			hub.AddOrder(o.Name, o.Content)
		}
	}
}
