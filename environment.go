package main

import (
	"log"
	"net/http"
	"time"
)

const (
	MODE_CMD  Mode = 1
	MODE_HTTP Mode = 2
)

type Mode int

type Environment struct {
	WorkerNum int
	Base      string
	DbFile    string
	Mode      Mode

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
