package server

import (
	"context"
	"fmt"
	"time"
)

// 运行的任务
type Order struct {
	Task

	Id      ID     `json:",omitempty"`
	Job     string `json:",omitempty"`
	Dirver  string `json:",omitempty"`
	AddTime int64  `json:",omitempty"`
	Retry   uint   `json:",omitempty"`
	PrevId  ID     `json:",omitempty"`

	ctx    context.Context
	cancel context.CancelFunc

	err    error
	status int
	resp   []byte

	startTime time.Time

	attr []any
}

func (s *Server) dirver(id ID, ctx context.Context) error {
	defer func() {
		s.store_order_del(id)
	}()

	o := s.store_order_get(id)
	if o == nil {
		s.log.Error("store_order_get not found", "id", id, "err", NotFound)

		return NotFound
	}

	o.attr = append(o.attr, "id", o.Id, "url", o.Task.Url)

	s.l.Lock()
	s.now = time.Now()
	o.startTime = s.now
	d, ok := s.cfg.Dirver[o.Dirver]
	var timeout uint = s.cfg.Timeout
	s.l.Unlock()

	if ok {
		if o.Task.Timeout > 0 {
			timeout = o.Task.Timeout
		}

		o.ctx, o.cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer o.cancel()

		d.run(o)
	} else {
		o.err = DirverNotFound
	}

	s.l.Lock()
	s.now = time.Now()
	now := s.now
	s.l.Unlock()

	s.logTask(now, o)

	if o.err != nil {
		if o.Retry < o.Task.Retry {
			o.Retry++

			var sec uint = 1

			if o.Task.Interval > 0 {
				sec = o.Task.Interval
			}

			o.Task.RunAt = now.Unix() + int64(sec)

			o.PrevId = o.Id
			o.Id = 0

			s.store_order_add(o)

			s.timer.push(int64(o.Task.RunAt), o.Id)
		}
	}

	return o.err
}

func (s *Server) logTask(now time.Time, o *Order) {
	runTime := now.Sub(o.startTime).Seconds()

	kv := append(o.attr, "cost", logCost(runTime), "status", o.Status)

	resp := string(o.resp)
	if resp == "" {
		resp = "-"
	}

	if o.err != nil {
		kv = append(kv, "err", o.err)

		if o.Retry > 0 {
			kv = append(kv, "retry", o.Retry)
		}

		s.log.Error(resp, kv...)
	} else {
		s.log.Info(resp, kv...)
	}
}

func logCost(ts float64) string {
	if ts >= 10 {
		return fmt.Sprintf("%ds", int(ts))
	}

	return fmt.Sprintf("%.2fs", ts)
}
