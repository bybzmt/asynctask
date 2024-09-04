package server

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sync"
	"time"
)

var Empty = errors.New("empty")
var NotFound = errors.New("NotFound")
var TaskError = errors.New("TaskError")
var DirverNotFound = errors.New("DirverNotFound")

func fmtId(id any) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, id)
	return buf.Bytes()
}

func copyMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))

	for k, v := range src {
		dst[k] = v
	}

	return dst
}

type logdb struct {
	l        sync.Mutex
	dir      string
	filefmt  string
	nextId   ID
	cache    *list.List
	cIndexes map[ID]*list.Element
	writeNum int
	fs       map[uint16]*os.File
	oldest   int
}

const fsIdxMask uint64 = 0x0000_ffff_0000_0000
const seekMask uint64 = 0x0000_0000_ffff_ffff

const fileMax = 1024 * 1024 * 100

const (
	ACTION_ADD = 1
	ACTION_DEL = 2
)

type dbItem struct {
	Order
	Action uint
}

func (db *logdb) cache_set(o *Order) {
	if ele, ok := db.cIndexes[o.Id]; ok {
		ele.Value = o
	} else {
		ele := db.cache.PushBack(o)
		db.cIndexes[o.Id] = ele

		if db.cache.Len() > 10 {
			ele := db.cache.Front()
			db.cache.Remove(ele)

			do := ele.Value.(*Order)
			delete(db.cIndexes, do.Id)
		}
	}
}

func (db *logdb) cache_get(id ID) *Order {
	if e, ok := db.cIndexes[id]; ok {
		return e.Value.(*Order)
	}
	return nil
}

func (db *logdb) cache_del(id ID) {
	if e, ok := db.cIndexes[id]; ok {
		o := e.Value.(*Order)
		delete(db.cIndexes, o.Id)
	}
}

func (db *logdb) file_open(fsidx uint16) *os.File {
	if f, ok := db.fs[fsidx]; ok {
		return f
	}

	name := fmt.Sprintf(db.filefmt, fsidx)

	nf, err := os.OpenFile(path.Join(db.dir, name), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	db.fs[fsidx] = nf

	return nf
}

func (db *logdb) item_read(id ID) *dbItem {

	fsidx := (uint64(id) & fsIdxMask) >> 32
	seek := uint64(id) & seekMask

	f := db.file_open(uint16(fsidx))

	if _, err := f.Seek(int64(seek), io.SeekStart); err != nil {
		panic(err)
	}

	dec := json.NewDecoder(f)

	item := &dbItem{}

	err := dec.Decode(item)
	if err != nil {
		panic(err)
	}

	return item
}

func (db *logdb) item_write(item *dbItem) {
	id := item.Id

	fsidx := (uint64(id) & fsIdxMask) >> 32

	f := db.file_open(uint16(fsidx))

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		panic(err)
	}

	dec := json.NewEncoder(f)

	if err := dec.Encode(&item); err != nil {
		panic(err)
	}

	seek, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		panic(err)
	}

	db.nextId = ID((fsidx << 32) | uint64(seek))
	db.writeNum++

	if db.writeNum > 100 {
		if err := f.Sync(); err != nil {
			panic(err)
		}
	}
}

func (db *logdb) tick_check() {
	db.l.Lock()
	defer db.l.Unlock()

	fsidx := (uint64(db.nextId) & fsIdxMask) >> 32

	f := db.file_open(uint16(fsidx))
	seek, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		panic(err)
	}

	if seek > fileMax {
		fsidx++
		if fsidx > 0xffff {
			fsidx = 1
		}

		db.nextId = ID(fsidx<<32 | uint64(seek))
	}

	for fsidx, f := range db.fs {
		if err := f.Sync(); err != nil {
			panic(err)
		}

		f.Close()

		delete(db.fs, fsidx)
	}
}

func (db *logdb) removeOldFiles() {
	files, err := os.ReadDir(db.dir)
	if err != nil {
		panic(err)
	}

	var fsidxs []uint16

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		var fsidx uint16
		_, err := fmt.Sscanf(file.Name(), db.filefmt, &fsidx)
		if err == nil && fsidx > 0 {
			fsidxs = append(fsidxs, fsidx)
		}
	}

	slices.Sort(fsidxs)

	nowidx := (uint64(db.nextId) & fsIdxMask) >> 32
	oldest := time.Now().Add(time.Hour * 24 * time.Duration(db.oldest))

	for _, fsidx := range fsidxs {
		if fsidx == uint16(nowidx) {
			continue
		}

		name := path.Join(db.dir, fmt.Sprintf(db.filefmt, fsidx))

		if fi, err := os.Stat(name); err == nil {
			if fi.ModTime().Before(oldest) {
				os.Remove(name)
			}
		}
	}
}

func (db *logdb) recoverIds() (ids map[ID]struct{}, maxId ID) {
	files, err := os.ReadDir(db.dir)
	if err != nil {
		panic(err)
	}

	var fsidxs []uint16

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		var fsidx uint16
		_, err := fmt.Sscanf(file.Name(), db.filefmt, &fsidx)
		if err == nil && fsidx > 0 {
			fsidxs = append(fsidxs, fsidx)
		}
	}

	slices.Sort(fsidxs)

	ids = make(map[ID]struct{})

	for _, fsidx := range fsidxs {
		name := fmt.Sprintf(db.filefmt, fsidx)

		nf, err := os.OpenFile(path.Join(db.dir, name), os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}

		dec := json.NewDecoder(nf)

		for dec.More() {
			item := &dbItem{}

			err := dec.Decode(item)
			if err != nil {
				panic(err)
			}

			if item.Id > maxId {
				maxId = item.Id
			}

			if item.Action == ACTION_ADD {
				ids[item.Id] = struct{}{}
			} else if item.Action == ACTION_DEL {
				delete(ids, item.Id)
			} else {
				panic("unknow action")
			}
		}

		nf.Close()
	}

	return ids, maxId
}

func (s *Server) store_order_get(id ID) *Order {
	s.db.l.Lock()
	defer s.db.l.Unlock()

	if o := s.db.cache_get(id); o != nil {
		return o
	}

	item := s.db.item_read(id)

	if item.Id != id {
		panic(fmt.Sprintf("expect id=%d got=%d", id, item.Order.Id))
	}

	if item.Action != ACTION_ADD {
		panic(fmt.Sprintf("item id=%d action=%d", id, item.Action))
	}

	return &item.Order
}

func (s *Server) store_order_del(id ID) {
	s.db.l.Lock()
	defer s.db.l.Unlock()

	item := &dbItem{
		Order: Order{
			Id: id,
		},
		Action: ACTION_DEL,
	}

	s.db.item_write(item)

	s.db.cache_del(id)
}

func (s *Server) store_order_add(o *Order) {
	s.db.l.Lock()
	defer s.db.l.Unlock()

	if o.Id != 0 {
		panic("order id not empty")
	}

	o.Id = s.db.nextId

	item := &dbItem{
		Order:  *o,
		Action: ACTION_ADD,
	}

	s.db.item_write(item)

	s.db.cache_set(o)
}

func (s *Server) store_init() error {
	s.db.l.Lock()

	s.db.cache = list.New()
	s.db.cIndexes = make(map[ID]*list.Element)
	s.db.fs = make(map[uint16]*os.File)

	ids, maxId := s.db.recoverIds()

	if maxId > 0 {
		fsidx := uint16((uint64(maxId) & fsIdxMask) >> 32)

		f := s.db.file_open(fsidx)

		seek2, err := f.Seek(0, io.SeekEnd)
		if err != nil {
			panic(err)
		}

		s.db.nextId = ID((uint64(fsidx) << 32) | uint64(seek2))
	} else {
		s.db.nextId = ID((uint64(1) << 32) | uint64(0))
	}

	s.db.l.Unlock()

	for id := range ids {
		o := s.store_order_get(id)
		s.orderAdd(o)
	}

	return nil
}

func json_encode(val any) string {
	out, err := json.Marshal(val)
	if err != nil {
		return ""
	}

	return string(out)
}
