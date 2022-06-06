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

	s.log.Println("[Info] redis init")
	defer s.log.Println("[Info] redis close")

	db, _ := strconv.Atoi(s.cfg.RedisDb)

	s.redis = redis.NewClient(&redis.Options{
		Addr:     s.cfg.RedisHost,
		Password: s.cfg.RedisPwd,
		DB:       db,
	})

	for s.running {
		if s.WaitNum > int(s.cfg.MaxTask) {
			time.Sleep(time.Second * 3)
			continue
		}

		out, err := s.redis.BLPop(time.Second*10, s.cfg.RedisKey).Result()

		if err != nil {
			s.log.Println("[Debug] redis list empty.", err.Error())
			time.Sleep(time.Second * 3)
		} else {
			o := Order{}
			err = json.Unmarshal([]byte(out[1]), &o)
			if err != nil {
				s.log.Println("[Debug] redis data Unmarshal error:", err.Error())
			} else {
				ok := s.AddOrderRel(&o)
				if !ok {
					out, _ := json.Marshal(&o)
					s.log.Println("[Info] add Task Fail", out)
				}
			}
		}
	}
}

func (s *Scheduler) redis_add(o *Order) bool {
	out, _ := json.Marshal(&o)
	s.redis.LPush(s.cfg.RedisKey, out)
	return true
}

func (s *Scheduler) saveToRedis() {
	s.jobs.Each(func(j *Job) {
		if j.Len() > 0 {
			ele := j.Tasks.Front()
			for ele != nil {
				t := ele.Value.(*taskMini)

				row := Order{
					Id:       t.Id,
					Parallel: j.parallel_abs,
					Name:     j.Name,
					Params:   t.Params,
					AddTime:  t.AddTime,
				}
				s.redis_add(&row)

				ele = ele.Next()
			}
		}
	})
}
