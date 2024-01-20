package scheduler

import (
	"context"
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
	CloseWait uint

	Jobs []*Job

	Groups map[string]*Group

	Dirver Dirver
	Log    Logger
}

type Group struct {
	WorkerNum uint32
	Note      string
}

type Job struct {
	Pattern  string
	Group    string
	Priority int32  //权重系数
	Parallel uint32 //并发数
}

type ID uint64

type Task struct {
	Id  ID
	Job string
}

type Dirver interface {
	Run(ID, context.Context) error
}

type DirverFunc func(ID, context.Context) error

func (f DirverFunc) Run(id ID, ctx context.Context) error {
	return f(id, ctx)
}

type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
