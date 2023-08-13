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

