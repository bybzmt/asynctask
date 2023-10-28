package scheduler

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	MODE_HTTP              Mode = 1
	MODE_HTTP_OVER_FASTCGI      = 4

	MODE_CLI          = 2
	MODE_CLI_OVER_SSH = 8
)

type Mode uint32
type ID uint32

type Config struct {
	//默认工作线程数量
	WorkerNum uint32
	//默认并发数
	Parallel uint32

	Client *http.Client

	Log logrus.FieldLogger
	Db  *bolt.DB
}

type Task struct {
	Cmd  string   `json:"cmd,omitempty"`
	Args []string `json:"args,omitempty"`

	Url    string            `json:"url,omitempty"`
	Method string            `json:"method,omitempty"`
	Header map[string]string `json:"header,omitempty"`
	Form   map[string]string `json:"form,omitempty"`
	Body   json.RawMessage   `json:"body,omitempty"`

	Timer    uint   `json:"timer,omitempty"`
	Timeout  uint   `json:"timeout,omitempty"`
	Hold     string `json:"hold,omitempty"`
	Code     int    `json:"code,omitempty"`
	Retry    uint   `json:"retry,omitempty"`
	RetrySec uint   `json:"retrySec,omitempty"`
}

type TaskBase struct {
	Timeout    uint //最大超时时间
	CmdBase    string
	CmdEnv     map[string]string
	CmdDir     string //工作目录
	HttpBase   string
	HttpHeader map[string]string
}

func (b *TaskBase) init() {
	if b.HttpHeader == nil {
		b.HttpHeader = make(map[string]string)
	}
	if b.CmdEnv == nil {
		b.CmdEnv = make(map[string]string)
	}
}

type GroupConfig struct {
	Id        ID
	WorkerNum uint32
	Note      string
}

type JobConfig struct {
	TaskBase
	Name     string
	Note     string
	GroupId  ID
	Priority int32  //权重系数
	Parallel uint32 //默认并发数
}

type schedulerConfig struct {
	TaskNextId ID
	WorkerId   ID
}
