package main

import (
	"time"
)

type Task struct {
	job    *Job
	worker *Worker

	Id     uint
	Params []string
	Status int
	Msg    string

	AddTime   time.Time
	StartTime time.Time
	EndTime   time.Time
}
