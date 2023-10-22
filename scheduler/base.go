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

func copyMap(src map[string]string) map[string]string {
    dst := make(map[string]string, len(src))

    for k, v := range src {
        dst[k] = v
    }

    return dst
}

func copyTaskBase(src TaskBase) (dst TaskBase) {
    dst = src
    dst.CliEnv = copyMap(src.CliEnv)
    dst.HttpHeader = copyMap(src.HttpHeader)
    return
}
