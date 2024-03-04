package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr    string
	Pwd     string
	Db      int
	Key     string
	Disable bool
}

func (c *RedisConfig) checkConfig() error {
	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Pwd,
		DB:       c.Db,
	})

	ctx := context.Background()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *RedisConfig) RedisLen() int64 {
	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Pwd,
		DB:       c.Db,
	})

	ctx := context.Background()

	a, _ := client.LLen(ctx, c.Key).Result()

	return a
}

func (c *RedisConfig) RedisRun(s *Server) {
	s.log.Println("redis init")
	defer s.log.Println("redis close")

	if c.Disable {
		return
	}

	s.l.Lock()

	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Pwd,
		DB:       c.Db,
	})

	ctx := s.ctx

	s.l.Unlock()

	go func() {
		<-ctx.Done()

		client.Close()
	}()

	for {
		out, err := client.BLPop(ctx, time.Second*5, c.Key).Result()

		if err != nil {
			if err.Error() == "redis: nil" {
				s.log.Debugln("redis", c.Key, "empty")
				time.Sleep(time.Second)
			} else {
				s.log.Debugln("redis", c.Key, "err:", err)
				return
			}
		} else {
			data := out[1]

			s.log.Debugln("redis task:", data)

			t := Task{}
			err = json.Unmarshal([]byte(data), &t)
			if err != nil {
				s.log.Warnln("redis data Unmarshal error:", err.Error(), data)
			} else {
				err := s.TaskAdd(&t)
				if err != nil {
					s.log.Warnln("redis add Task Fail", data)
				}
			}
		}
	}
}
