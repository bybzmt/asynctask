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
	//工作线程数量
	WorkerNum int
	//任务Base
	Base string
	//任务模式cmd/http
	Mode Mode
	//默认并发数
	Parallel uint
	//任务超时时间
	TaskTimeout time.Duration

	LogFile string
	DbFile  string
	MaxTask uint

	RedisHost string
	RedisPwd  string
	RedisDb   string
	RedisKey  string

	Client *http.Client

	//统计周期
	StatTick time.Duration
	StatSize int
}

func (c *Config) Init(mode string, timeout int) {
	c.TaskTimeout = time.Second * time.Duration(timeout)

	tr := &http.Transport{
		MaxIdleConnsPerHost: c.WorkerNum,
	}

	c.Client = &http.Client{
		Transport: tr,
		Timeout:   c.TaskTimeout,
	}

	if mode == "cmd" {
		c.Mode = MODE_CMD
	} else {
		c.Mode = MODE_HTTP
	}

	c.StatTick = time.Second * 1
	c.StatSize = 30

	if c.DbFile == "" {
		c.DbFile = "./asynctask.db"
	}

	if c.WorkerNum < 1 {
		c.WorkerNum = 10
	}

	if c.Parallel < 1 {
		c.Parallel = 5
	}
}
