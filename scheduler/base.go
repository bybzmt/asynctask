package scheduler

import (
	"errors"
	"fmt"
	"strconv"

	bolt "go.etcd.io/bbolt"
)

var Empty = errors.New("empty")
var NotFound = errors.New("NotFound")
var TaskError = errors.New("TaskError")

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
	Task     string
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
	return fmt.Sprintf("%015d", id)
}

func atoiId(key []byte) ID {
	id, _ := strconv.Atoi(string(key))
	return ID(id)
}

func copyBase(src, dst *TaskBase) {
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
