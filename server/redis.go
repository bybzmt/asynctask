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
	s.log.Info("redis init")
	defer s.log.Info("redis close")

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
			switch err.Error() {
			case "redis: nil":
				s.log.Debug("redis key empty", "addr", c.Addr, "key", c.Key)
			case "redis: client is closed":
				fallthrough
			case "context canceled":
				s.log.Debug("redis close", "addr", c.Addr, "key", c.Key, "err", err)
				return
			default:
				s.log.Debug("redis error", "addr", c.Addr, "key", c.Key, "err", err)
				time.Sleep(time.Second)
			}
		} else {
			data := out[1]

			t := Task{}
			err = json.Unmarshal([]byte(data), &t)
			if err != nil {
				s.log.Error("redis data Unmarshal error", "err", err.Error(), "data", data)
			} else {
				err := s.TaskAdd(&t)
				if err != nil {
					s.log.Error("redis add Task Fail", "data", data)
				}
			}
		}
	}
}
