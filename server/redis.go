package server

import (
	"asynctask/scheduler"
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"strconv"
	"time"
)

type RedisConfig struct {
	Addr string
	Pwd  string
	Db   string
	Key  string
}

func (s *Server) initRedis() error {
    if s.Redis.Key == "" {
        return nil
    }

	redis := s.getRedis()

	_, err := redis.LLen(s.Redis.Key).Result()

	return err
}

func (s *Server) getRedis() *redis.Client {

	_db, _ := strconv.Atoi(s.Redis.Db)

	redis := redis.NewClient(&redis.Options{
		Addr:     s.Redis.Addr,
		Password: s.Redis.Pwd,
		DB:       _db,
	})

	return redis
}

func (s *Server) RedisRun(ctx context.Context) {
	s.Scheduler.Log.Println("[Info] redis init")
	defer s.Scheduler.Log.Println("[Info] redis close")

	redis := s.getRedis()
	redis = redis.WithContext(ctx)

	for {
		out, err := redis.BLPop(time.Second*5, s.Redis.Key).Result()

		exit := false

		select {
		case <-ctx.Done():
			exit = true
		default:
		}

		if err != nil {
			s.Scheduler.Log.Debugln("redis list empty.", err.Error())

			if exit {
				return
			}

			time.Sleep(time.Second)
		} else {
			data := out[1]

			o := scheduler.Task{}
			err = json.Unmarshal([]byte(data), &o)
			if err != nil {
				s.Scheduler.Log.Warnln("redis data Unmarshal error:", err.Error(), data)
			} else {
				err := s.Scheduler.AddTask(&o)
				if err != nil {
					s.Scheduler.Log.Warnln("redis add Task Fail", data)
				}
			}

			if exit {
				return
			}
		}
	}
}
