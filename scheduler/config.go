package scheduler

import (
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"net/http"
    "time"
)

const (
	MODE_HTTP Mode = 1
	MODE_HTTP_OVER_FASTCGI = 4

	MODE_CMD = 2
	MODE_CMD_OVER_SSH = 8
)

type Mode uint32
type ID uint32

type Config struct {
	//默认工作线程数量
	WorkerNum int
	//默认并发数
	Parallel uint

	Client *http.Client

	Log logrus.FieldLogger
	Db  *bolt.DB
}

type Task struct {
	Name    string    `json:"name,omitempty"`
	Trigger uint      `json:"trigger,omitempty"`
	Http    *TaskHttp `json:"http,omitempty"`
	Cli     *TaskCli  `json:"cli,omitempty"`
	Timeout uint      `json:"timeout,omitempty"`
	Hold    string    `json:"hold,omitempty"`
	Id      uint      `json:",omitempty"`
	AddTime uint      `json:",omitempty"`
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
	Base *TaskBase

	Status int
	Msg    string
	Err    error

	AddTime   time.Time
	StartTime time.Time
	StatTime  time.Time
	EndTime   time.Time
}

type TaskBase struct {
	Mode   Mode
	Timeout    uint //默认超时时间
	CmdBase    string
	CmdEnv     map[string]string
	HttpBase   string
	HttpHeader map[string]string
}

func (b *TaskBase) init() {
    b.HttpHeader = make(map[string]string)
    b.CmdEnv = make(map[string]string)
}

type GroupJobConfig struct {
	GroupId ID
	Note     string
	Priority int
	Parallel uint32
}

type RouterConfig struct {
	TaskBase
	Match  string
	Note   string
    Groups []GroupJobConfig
    Sort   int
    Used   bool
}

type GroupConfig struct {
	Parallel  uint32 //默认并发数
	WorkerNum uint32
	Weight    uint32
	Note      string
}

type JobConfig struct {
	Note     string
	Priority int
	Parallel uint32 //默认并发数
}
