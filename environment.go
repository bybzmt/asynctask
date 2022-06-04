package main

import (
	"net/http"
	"time"
)

const (
	MODE_HTTP Mode = iota
	MODE_CMD
)

type Mode int

type Config struct {
	WorkerNum int
	Base      string
	Mode      Mode
	Parallel  uint

	LogFile string
	DbFile  string
	MaxMem  uint

	RedisHost string
	RedisPwd  string
	RedisDb   string
	RedisKey  string

	Client      *http.Client
	TaskTimeout time.Duration

	//统计周期
	StatTick time.Duration
	StatSize int
}

func (a *Config) Init(mode string, timeout int) {
	a.TaskTimeout = time.Second * time.Duration(timeout)

	tr := &http.Transport{
		MaxIdleConnsPerHost: a.WorkerNum,
	}

	a.Client = &http.Client{
		Transport: tr,
		Timeout:   a.TaskTimeout,
	}

	if mode == "cmd" {
		a.Mode = MODE_CMD
	} else {
		a.Mode = MODE_HTTP
	}

	a.StatTick = time.Second * 1
	a.StatSize = 30
}
