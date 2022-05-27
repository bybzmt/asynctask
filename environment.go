package main

import (
	"log"
	"net/http"
	"time"
)

const (
	MODE_HTTP Mode = iota
	MODE_CMD
)

type Mode int

type Environment struct {
	WorkerNum int
	Base      string
	DbFile    string
	Mode      Mode
	Parallel  uint

	Log *log.Logger

	Client  *http.Client
	Timeout time.Duration

	//统计周期
	StatTick time.Duration
	StatSize int
}

func (a *Environment) Init(workerNum int, base string, timeout int, out *log.Logger) *Environment {
	a.WorkerNum = workerNum
	a.Base = base

	a.Log = out
	a.Parallel = 5
	a.Timeout = time.Second * time.Duration(timeout)

	tr := &http.Transport{
		MaxIdleConnsPerHost: a.WorkerNum,
	}

	a.Client = &http.Client{
		Transport: tr,
		Timeout:   a.Timeout,
	}

	return a
}
