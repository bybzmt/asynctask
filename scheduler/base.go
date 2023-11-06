package scheduler

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

func db_keys(keys []string) (buckets []string, key string) {
	l := len(keys)
	buckets = keys[:l-1]
	key = keys[l-1]
	return
}

func db_fetch(db *bolt.DB, v any, key ...string) error {
	buckets, k := db_keys(key)

	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, buckets...)
		if b == nil {
			return nil
		}

		out := b.Get([]byte(k))
		if out != nil {
			return json.Unmarshal(out, v)
		}
		return nil
	})
}

func db_put(db *bolt.DB, val any, key ...string) error {
	v, err := json.Marshal(val)
	if err != nil {
		return err
	}

	buckets, k := db_keys(key)

	return db.Update(func(tx *bolt.Tx) error {
		b, err := getBucketMust(tx, buckets...)
		if err != nil {
			return err
		}

		return b.Put([]byte(k), v)
	})
}

func db_push(db *bolt.DB, t any, bucket ...string) error {
	val, err := json.Marshal(t)
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b, err := getBucketMust(tx, bucket...)
		if err != nil {
			return err
		}

		id, err := b.NextSequence()
		if err != nil {
			return err
		}

		return b.Put([]byte(fmtId(id)), val)
	})
}

func db_del(db *bolt.DB, key ...string) error {
	buckets, k := db_keys(key)

	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, buckets...)
		if b == nil {
			return nil
		}

		return b.Delete([]byte(k))
	})
}

func db_shift(db *bolt.DB, out any, bucket ...string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, bucket...)
		if b == nil {
			return Empty
		}

		key, val := b.Cursor().First()

		if key == nil {
			return Empty
		}

        err := b.Delete(key)
        if err != nil {
            return err
        }

		return json.Unmarshal(val, &out)
	})
}

func db_get_buckets(db *bolt.DB, bucket ...string) (buckets []string) {
	db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, bucket...)
		if b == nil {
			return nil
		}

		b.ForEachBucket(func(k []byte) error {
			buckets = append(buckets, string(k))
			return nil
		})

		return nil
	})

	return
}

func db_bucket_del(db *bolt.DB, bucket ...string) error {
	buckets, k := db_keys(bucket)

	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, buckets...)
		if b == nil {
			return nil
		}

		return b.DeleteBucket([]byte(k))
	})
}

func db_getall(db *bolt.DB, fn func(k, v []byte) error, bucket ...string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, bucket...)
		if b == nil {
			return nil
		}

		b.ForEach(fn)

		return nil
	})
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

func jobAppend(j, at *job) {
	at.next.prev = j
	j.next = at.next
	j.prev = at
	at.next = j
}

func jobRemove(j *job) {
	j.prev.next = j.next
	j.next.prev = j.prev
	j.next = nil
	j.prev = nil
}

func jobMoveBefore(j, x *job) {
	if j == x {
		return
	}

	jobRemove(j)
	jobAppend(j, x.prev)
}
