package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

func (s *Scheduler) redis_init() {
	if s.cfg.RedisHost == "" {
		return
	}

	s.log.Println("[Info] use redis")

	db, _ := strconv.Atoi(*redis_db)

	client := redis.NewClient(&redis.Options{
		Addr:     s.cfg.RedisHost,
		Password: s.cfg.RedisPwd,
		DB:       db,
	})
	defer client.Close()

	for s.running {
		if s.memFull {
			time.Sleep(time.Second * 3)
			continue
		}

		out, err := client.BLPop(time.Second*10, s.cfg.RedisKey).Result()

		if err != nil {
			s.log.Println("[Debug] redis list empty.", err.Error())
			time.Sleep(time.Second * 3)
		} else {
			o := Order{}
			err = json.Unmarshal([]byte(out[1]), &o)
			if err != nil {
				s.log.Println("[Debug] redis data Unmarshal error:", err.Error())
			} else {
				ok := s.AddOrder(&o)
				if !ok {
					out, _ := json.Marshal(&o)
					s.log.Println("[Info] add Task Fail", out)
				}
			}
		}
	}
}
