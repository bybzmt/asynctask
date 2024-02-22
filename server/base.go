package server

import (
	"asynctask/scheduler"
)

type ID = scheduler.ID
type Job = scheduler.Job
type Group = scheduler.Group
type task = scheduler.Task

type DirverType uint

const (
	DIRVER_HTTP    DirverType = 1
	DIRVER_CGI                = 2
	DIRVER_FASTCGI            = 3
)

type Config struct {
	//默认组
	Group string
	//默认工作线程数量
	WorkerNum uint32
	//默认并发数
	Parallel uint32
	//默认超时
	Timeout uint
	//空闲数量
	JobsMaxIdle uint
	//关闭等待
	CloseWait uint `json:",omitempty"`

	HttpAddr   string `json:",omitempty"`
	HttpEnable bool   `json:",omitempty"`

	Jobs   []*Job             `json:",omitempty"`
	Routes []*Route           `json:",omitempty"`
	Groups map[string]*Group  `json:",omitempty"`
	Dirver map[string]*Dirver `json:",omitempty"`
	Redis  []RedisConfig      `json:",omitempty"`
	Crons  []CronTask         `json:",omitempty"`
}

type Task struct {
	Method string            `json:"method,omitempty"`
	Url    string            `json:"url,omitempty"`
	Header map[string]string `json:"header,omitempty"`
	Body   []byte            `json:"body,omitempty"`

	RunAt    int64  `json:"runat,omitempty"`
	Timeout  uint   `json:"timeout,omitempty"`
	Hold     string `json:"hold,omitempty"`
	Status   int    `json:"status,omitempty"`
	Retry    uint   `json:"retry,omitempty"`
	Interval uint   `json:"interval,omitempty"`
}

