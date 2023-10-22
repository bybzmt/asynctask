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

	Name     string `json:"name,omitempty"`
	Timer    uint   `json:"timer,omitempty"`
	Timeout  uint   `json:"timeout,omitempty"`
	Hold     string `json:"hold,omitempty"`
	Code     int    `json:"code,omitempty"`
	Retry    uint   `json:"retry,omitempty"`
	RetrySec uint   `json:"retrySec,omitempty"`
	Id       uint   `json:",omitempty"`
	AddTime  uint   `json:",omitempty"`
}

type TaskBase struct {
	Mode       Mode
	Timeout    uint //最大超时时间
	CliBase    string
	CliEnv     map[string]string
	CliDir     string //工作目录
	HttpBase   string
	HttpHeader map[string]string
}

func (b *TaskBase) init() {
	if b.HttpHeader == nil {
		b.HttpHeader = make(map[string]string)
	}
	if b.CliEnv == nil {
		b.CliEnv = make(map[string]string)
	}
}

type TaskConfig struct {
	JobConfig
	TaskBase
	Id      ID
	Match   string
	Note    string
	GroupId ID
	Sort    int
	Used    bool
}

type GroupConfig struct {
	Id        ID
	WorkerNum uint32
	Note      string
}

type JobConfig struct {
	Priority int32  //权重系数
	Parallel uint32 //默认并发数
}

type schedulerConfig struct {
	TaskNextId ID
	WorkerId   ID
}
