package server

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	cache    *list.List
	cIndexes map[ID]*list.Element
	writeNum int
	fs       map[uint16]*os.File
	fid      uint16
	oldest   int
}

const fidMask uint64 = 0x0000_ffff_0000_0000
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

		if db.cache.Len() > 1000 {
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

func (db *logdb) file_open(fid uint16) *os.File {
	if f, ok := db.fs[fid]; ok {
		return f
	}

	name := path.Join(db.dir, fmt.Sprintf(db.filefmt, fid))

	nf, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	db.fs[fid] = nf

	return nf
}

func (db *logdb) item_read(id ID) *dbItem {

	fid := (uint64(id) & fidMask) >> 32
	seek := uint64(id) & seekMask

	f := db.file_open(uint16(fid))

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
	f := db.file_open(db.fid)

	seek, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		panic(err)
	}

	if item.Id == 0 {
		item.Id = ID(uint64(db.fid)<<32 | uint64(seek))
	}

	dec := json.NewEncoder(f)

	if err := dec.Encode(&item); err != nil {
		panic(err)
	}

	db.writeNum++

	if db.writeNum > 100 {
		if err := f.Sync(); err != nil {
			panic(err)
		}
	}
}

func (db *logdb) removeOldFiles() {
	files, err := os.ReadDir(db.dir)
	if err != nil {
		panic(err)
	}

	var fids []uint16

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		var fid uint16
		_, err := fmt.Sscanf(file.Name(), db.filefmt, &fid)
		if err == nil && fid > 0 {
			fids = append(fids, fid)
		}
	}

	slices.Sort(fids)

	oldest := time.Now().Add(-(time.Hour * 24 * time.Duration(db.oldest)))

	for _, fid := range fids {
		if fid == uint16(db.fid) {
			continue
		}

		name := path.Join(db.dir, fmt.Sprintf(db.filefmt, fid))

		if fi, err := os.Stat(name); err == nil {
			if fi.ModTime().Before(oldest) {
				os.Remove(name)
			}
		}
	}
}

func (db *logdb) recoverFileOrders(fid uint16) (ids map[ID]struct{}, err error) {
	ids = make(map[ID]struct{})

	name := path.Join(db.dir, fmt.Sprintf(db.filefmt, fid))

	nf, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer nf.Close()

	dec := json.NewDecoder(nf)

	for dec.More() {
		item := &dbItem{}

		err = dec.Decode(item)
		if err != nil {
			return
		}

		if item.Action == ACTION_ADD {
			ids[item.Id] = struct{}{}
		} else if item.Action == ACTION_DEL {
			delete(ids, item.Id)
		} else {
			err = fmt.Errorf("unknow action:%d", item.Action)
			return
		}
	}

	return
}

func (db *logdb) recoverOrders(log *slog.Logger) map[ID]struct{} {
	files, err := os.ReadDir(db.dir)
	if err != nil {
		panic(err)
	}

	var fids []uint16

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		var fid uint16
		_, err := fmt.Sscanf(file.Name(), db.filefmt, &fid)
		if err == nil && fid > 0 {
			fids = append(fids, fid)
		}
	}

	slices.Sort(fids)

	ids := make(map[ID]struct{})

	var maxfid uint16

	skipErrFile := false

	for _, fid := range fids {
		if maxfid == 0 {
			maxfid = fid
		} else if maxfid+1 == fid {
			maxfid = fid
		}

		fids, err := db.recoverFileOrders(fid)

		if err != nil {
			skipErrFile = true

			log.Error("recoverFileOrders error skip", "fid", fid, "err", err)
		} else {
			skipErrFile = false

			for id := range fids {
				ids[id] = struct{}{}
			}
		}
	}

	if skipErrFile {
		maxfid++
	}

	for {
		if maxfid == 0 {
			maxfid = 1
		}

		f := db.file_open(maxfid)

		seek, err := f.Seek(0, io.SeekEnd)
		if err != nil {
			panic(err)
		}

		if seek > fileMax {
			maxfid++
			continue
		}

		break
	}

	db.fid = maxfid

	return ids
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
		panic("add order id need empty")
	}

	item := &dbItem{
		Order:  *o,
		Action: ACTION_ADD,
	}

	s.db.item_write(item)

	o.Id = item.Id

	s.db.cache_set(o)
}

func (s *Server) store_init() error {
	s.db.l.Lock()

	s.db.cache = list.New()
	s.db.cIndexes = make(map[ID]*list.Element)
	s.db.fs = make(map[uint16]*os.File)

	ids := s.db.recoverOrders(s.log)

	s.db.l.Unlock()

	for id := range ids {
		o := s.store_order_get(id)
		s.orderAdd(o)
	}

	return nil
}

func (s *Server) store_close() {
	s.db.l.Lock()
	defer s.db.l.Unlock()

	for idx, f := range s.db.fs {
		if err := f.Sync(); err != nil {
			panic(err)
		}

		f.Close()

		delete(s.db.fs, idx)
	}
}

func (s *Server) store_tick() {
	s.db.l.Lock()
	defer s.db.l.Unlock()

	f := s.db.file_open(s.db.fid)
	seek, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		panic(err)
	}

	for idx, f := range s.db.fs {
		if err := f.Sync(); err != nil {
			panic(err)
		}

		f.Close()

		delete(s.db.fs, idx)
	}

	if seek > fileMax {
		s.db.fid++
		if s.db.fid == 0 {
			s.db.fid = 1
		}

		s.db.removeOldFiles()
	}
}

func json_encode(val any) string {
	out, err := json.Marshal(val)
	if err != nil {
		return ""
	}

	return string(out)
}
