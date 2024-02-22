package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// 运行的任务
type Order struct {
	Id      ID     `json:",omitempty"`
	Job     string `json:",omitempty"`
	Task    Task   `json:",omitempty"`
	Dirver  string `json:",omitempty"`
	AddTime int64  `json:",omitempty"`
	Retry   uint   `json:",omitempty"`

	ctx    context.Context
	cancel context.CancelFunc

	err    error
	status int
	resp   []byte

	startTime time.Time

	fields map[string]any
}

func (s *Server) dirver(id ID, ctx context.Context) error {

	o := s.store_order_get(id)

	if o == nil {
		s.log.WithFields(map[string]any{
			"id":  id,
			"err": NotFound,
		}).Errorln("store_order_get not found")

		return NotFound
	}

	o.fields = map[string]any{
		"id":  o.Id,
		"job": o.Job,
	}

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

	del := true

	if o.err != nil {
		if o.Retry < o.Task.Retry {
			o.Retry++

			var sec uint = 1

			if o.Task.Interval > 0 {
				sec = o.Task.Interval
			}

			o.Task.RunAt = now.Unix() + int64(sec)

			s.store_order_put(o)

			s.timer.push(int64(o.Task.RunAt), o.Id)

			del = false
		}
	}

	if del {
		s.store_order_del(o.Id)
	}

	s.logTask(now, o)

	return o.err
}

func (s *Server) logTask(now time.Time, o *Order) {
	runTime := now.Sub(o.startTime).Seconds()

	o.fields["cost"] = logCost(runTime)
	o.fields["status"] = o.status

	if o.err != nil {
		o.fields["err"] = o.err

		if xx, err := json.Marshal(o.Task); err == nil {
			o.fields["task"] = string(xx)
		}

		if o.Retry > 0 {
			o.fields["retry"] = o.Retry
		}

		s.log.WithFields(o.fields).Errorf("%s", o.resp)
		return
	}

	s.log.WithFields(o.fields).Infof("%s", o.resp)
}

func logCost(ts float64) string {
	if ts >= 10 {
		return fmt.Sprintf("%ds", int(ts))
	}

	return fmt.Sprintf("%.2fs", ts)
}
