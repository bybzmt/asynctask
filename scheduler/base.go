package scheduler

import (
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// redis队列 json结构
type Task struct {
	Id      uint      `json:"id,omitempty"`
	Name    string    `json:"name,omitempty"`
	Trigger uint      `json:"trigger,omitempty"`
	Http    *TaskHttp `json:"http,omitempty"`
	Cli     *TaskCli  `json:"cli,omitempty"`
	Timeout uint      `json:"timeout,omitempty"`
	Hold    string    `json:"hold,omitempty"`
}

type TaskHttp struct {
	Method string            `json:"method,omitempty"`
	Url    string            `json:"url"`
	Header map[string]string `json:"header,omitempty"`
	Body   string            `json:"body,omitempty"`
	Get    map[string]string `json:"get,omitempty"`
	Post   map[string]string `json:"post,omitempty"`
	Json   string            `json:"json,omitempty"`
}

type TaskCli struct {
	Cmd    string   `json:"cmd"`
	Params []string `json:"params,omitempty"`
}

// 运行的任务
type order struct {
	Id     ID
	job    *job
	worker *worker

	Task *Task
    Base JobBase

	Status int
	Msg    string
	Err    error

	AddTime   time.Time
	StartTime time.Time
	StatTime  time.Time
	EndTime   time.Time
}

// 输出日志时间（为了不显示太长的小数)
type logSecond float64

func (l logSecond) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf("%.2f", l)
	return []byte(str), nil
}

// task 日志记录
type taskLog struct {
	Id       ID
	Name     string
	Params   []string
	Status   int
	WaitTime logSecond
	RunTime  logSecond
	Output   string
}

type JobBase struct {
	Timeout    uint //默认超时时间
	CmdBase    string
	CmdEnv     map[string]string
	HttpBase   string
	HttpHeader map[string]string
}

func (b *JobBase) init() {
    b.HttpHeader = make(map[string]string)
    b.CmdEnv = make(map[string]string)
}

type RouterConfig struct {
	JobBase
	Id     ID
	Match  string
	Name   string
	Groups []ID
	Weights []uint32
	Mode   Mode
    Sort   int
}

type GroupConfig struct {
	JobBase

	Parallel  uint32 //默认并发数
	WorkerNum uint32
	Weight    uint32
	Id        ID
	Match     string
	Name      string
}

type JobConfig struct {
	Id       ID
	Name     string
	Priority int
	Parallel uint32 //默认并发数
}

type bucketer interface {
	Bucket(key []byte) *bolt.Bucket
	CreateBucketIfNotExists(key []byte) (*bolt.Bucket, error)
}

func getBucketMust(bk bucketer, keys ...string) (*bolt.Bucket, error) {
	if len(keys) == 0 {
		panic(errors.New("keys empty"))
	}

	out := bk

	for _, key := range keys {
		t, err := out.CreateBucketIfNotExists([]byte(key))
		if err != nil {
			return nil, err
		}
		out = t
	}

	return out.(*bolt.Bucket), nil
}

func getBucket(bk bucketer, keys ...string) *bolt.Bucket {
	if len(keys) == 0 {
		panic(errors.New("keys empty"))
	}

	out := bk

	for _, key := range keys {
		t := out.Bucket([]byte(key))
		if t == nil {
			return nil
		}
		out = t
	}

	return out.(*bolt.Bucket)
}

func fmtId(id any) string {
	return fmt.Sprintf("%12d", id)
}

func copyBase(src, dst *JobBase) {
    if src.Timeout > 0 {
        dst.Timeout = src.Timeout
    }

    if src.CmdBase != "" {
        dst.CmdBase = src.CmdBase
    }

    if src.HttpBase != "" {
        dst.HttpBase = src.HttpBase
    }

    for k, v := range src.CmdEnv {
        dst.CmdEnv[k] = v
    }

    for k, v := range src.HttpHeader {
        dst.HttpHeader[k] = v
    }
}
