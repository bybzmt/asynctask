package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

type Environment struct {
	WorkerNum int
	BaseUrl   string
	Log       *log.Logger
	Info      *log.Logger
	Client    *http.Client
	timeout   time.Duration

	//统计周期
	StatTick time.Duration
	StatSize int
}

func (a *Environment) Init(workerNum int, baseurl string, out, err *log.Logger) *Environment {
	a.WorkerNum = workerNum
	a.BaseUrl = baseurl

	if out == nil {
		out = log.New(os.Stdout, "[Info] ", log.Ldate)
	}
	if err == nil {
		err = log.New(os.Stderr, "[Scheduler] ", log.LstdFlags)
	}

	a.Info = out
	a.Log = err

	a.timeout = time.Second * 300

	tr := &http.Transport{
		MaxIdleConnsPerHost: a.WorkerNum,
	}

	a.Client = &http.Client{
		Transport: tr,
		Timeout:   a.timeout,
	}

	return a
}
