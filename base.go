package main

import (
	"fmt"
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
	worker Worker

	Id     uint
	Params []string
	Status int
	Msg    string
	Err    error

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

//输出日志时间（为了不显示太长的小数)
type LogSecond float64

func (l LogSecond) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf("%.2f", l)
	return []byte(str), nil
}

//task 日志记录
type TaskLog struct {
	Id       uint
	Name     string
	Params   []string
	Status   int
	WaitTime LogSecond
	RunTime  LogSecond
	Output   string
}

//工作线程
type Worker interface {
	Exec(t *Task)
	Cancel()
	Run()
	Close()
}
