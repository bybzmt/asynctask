package main

import (
	"time"
)

type Task struct {
	job    *Job
	worker *Worker

	Id      int
	Content string
	Status  int
	Msg     string

	AddTime   time.Time
	StartTime time.Time
	EndTime   time.Time
}
