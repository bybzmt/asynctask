package server

import (
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

	r *redis.Client
}

func (c *RedisConfig) checkConfig() error {
	_db, _ := strconv.Atoi(c.Db)

	redis := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Pwd,
		DB:       _db,
	})

	_, err := redis.Ping().Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *RedisConfig) RedisRun(s *Server) {
	s.log.Println("[Info] redis init")
	defer s.log.Println("[Info] redis close")

	s.l.Lock()
	_db, _ := strconv.Atoi(c.Db)

	redis := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Pwd,
		DB:       _db,
	})

	redis = redis.WithContext(s.ctx)
	c.r = redis
	ctx := s.ctx

	s.l.Unlock()

	// _, err := redis.LLen(c.Key).Result()

	for {
		out, err := redis.BLPop(time.Second*5, c.Key).Result()

		if err != nil {
			s.log.Debugln("redis list empty.", c.Key, err.Error())

			select {
			case <-ctx.Done():
			default:
				time.Sleep(time.Second)
			}
		} else {
			data := out[1]

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

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
