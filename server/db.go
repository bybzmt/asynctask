package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	bolt "go.etcd.io/bbolt"
)

var Empty = errors.New("empty")
var NotFound = errors.New("NotFound")
var TaskError = errors.New("TaskError")
var DirverNotFound = errors.New("DirverNotFound")

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

func (s *Server) store_order_get(id ID) *Order {

	var t []byte

	err := s.db.View(func(tx *bolt.Tx) error {
		b := getBucket(tx, "tasks")
		if b == nil {
			return nil
		}

		t = b.Get([]byte(fmtId(id)))

		return nil
	})

	if err != nil {
		s.log.Warnln("store_order_get Error", err)
		return nil
	}

	if t == nil {
		return nil
	}

	out := new(Order)

	if err := json.Unmarshal(t, out); err != nil {
		s.log.Warnln("Order Unmarshal Error", err)
		return nil
	}

	return out
}

func (s *Server) store_order_del(id ID) {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, "tasks")
		if b == nil {
			return nil
		}

		return b.Delete([]byte(fmtId(id)))
	})

	if err != nil {
		s.log.Warnln("store_order_del error", err)
	}
}

func (s *Server) store_order_add(o *Order) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := getBucketMust(tx, "tasks")
		if err != nil {
			return err
		}

		id, err := b.NextSequence()
		if err != nil {
			return err
		}

		o.Id = ID(id)

		v, err := json.Marshal(o)
		if err != nil {
			return err
		}

		return b.Put([]byte(fmtId(o.Id)), v)
	})
}

func (s *Server) store_order_put(o *Order) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := getBucketMust(tx, "tasks")
		if err != nil {
			return err
		}

		v, err := json.Marshal(o)
		if err != nil {
			return err
		}

		return b.Put([]byte(fmtId(o.Id)), v)
	})
}

func json_encode(val any) string {
	out, err := json.Marshal(val)
	if err != nil {
		return ""
	}

	return string(out)
}
