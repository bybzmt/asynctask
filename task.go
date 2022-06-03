package main

import (
	"time"
)

//redis队列 json结构
type Order struct {
	Id       uint     `json:"id,omitempty"`
	Parallel uint     `json:"parallel,omitempty"`
	Name     string   `json:"name"`
	Params   []string `json:"params,omitempty"`
	AddTime  uint     `json:"add_time,omitempty"`
}

//运行的任务
type Task struct {
	job    *Job
	worker *Worker

	Id     uint
	Params []string
	Status int
	Msg    string

	AddTime   time.Time
	StartTime time.Time
	StatTime  time.Time
	EndTime   time.Time
}

//task去掉非必要字段，节省内存
type taskMini struct {
	Id      uint
	Params  []string
	AddTime uint
}

type taskMiniList struct {
	offset int
	next   *taskMiniList
	list   [150]taskMini
}

//task 日志记录
type TaskLog struct {
	Id       uint
	Name     string
	Params   []string
	Status   int
	WaitTime float64
	RunTime  float64
	Output   string
}
