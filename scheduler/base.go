package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

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

func atoiId(key []byte) ID {
	id, _ := strconv.Atoi(string(key))
	return ID(id)
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