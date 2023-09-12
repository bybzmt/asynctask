package scheduler

import (
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"net/http"
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

type OrderBase struct {
	Timeout    uint //默认超时时间
	CmdBase    string
	CmdEnv     map[string]string
	HttpBase   string
	HttpHeader map[string]string
}

func (b *OrderBase) init() {
    b.HttpHeader = make(map[string]string)
    b.CmdEnv = make(map[string]string)
}

type RouterConfig struct {
	OrderBase
	Match  string
	Note   string
	Groups []ID
	Weights []uint32
	Mode   Mode
    Sort   int
}

type GroupConfig struct {
	OrderBase

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
