package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	status int
	msg    string
	err    error

	startTime time.Time

	taskTxt string
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

	o.startTime = time.Now()

	s.l.Lock()
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
		o.status = -1
		o.err = DirverNotFound
	}

	del := true

	if o.err != nil {
		if o.Retry < o.Task.Retry {
			o.Retry++

			var sec uint = 1

			if o.Task.Interval > 0 {
				sec = o.Task.Interval
			}

			o.Task.RunAt = time.Now().Unix() + int64(sec)

			s.store_order_put(o)

			s.timer.push(int64(o.Task.RunAt), o.Id)

			del = false
		}
	}

	if del {
		s.store_order_del(o.Id)
	}

	s.logTask(o)

	return o.err
}

func (s *Server) logTask(o *Order) {
	now := time.Now()

	runTime := now.Sub(o.startTime).Seconds()

	logFields := map[string]any{
		"id":     o.Id,
		"job":    o.Job,
		"url":    o.taskTxt,
		"status": o.status,
		"cost":   logCost(runTime),
	}

	if o.err != nil {
		logFields["err"] = o.err

		if xx, err := json.Marshal(o.Task); err == nil {
			logFields["task"] = string(xx)
		}

		if o.Retry > 0 {
			logFields["retry"] = o.Retry
		}

		s.log.WithFields(logFields).Errorln(o.msg)
	} else {
		s.log.WithFields(logFields).Infoln(o.msg)
	}
}

func logCost(ts float64) string {
	if ts >= 10 {
		return fmt.Sprintf("%ds", int(ts))
	}

	return fmt.Sprintf("%.2fs", ts)
}

func (d *Dirver) run(o *Order) {
	switch d.Type {
	case DIRVER_HTTP:
		d.http.run(o)

	case DIRVER_CGI:
		d.Cgi.run(o)

	case DIRVER_FASTCGI:
		d.Fcgi.run(o)

	default:
		o.status = -1
		o.err = TaskError
	}
}

func (d *Dirver) init() error {

	switch d.Type {
	case DIRVER_HTTP:
		d.http = &dirverHttp{
			client: &http.Client{},
		}

	case DIRVER_CGI:

	case DIRVER_FASTCGI:

	default:
		return TaskError
	}

	return nil
}

type dirverHttp struct {
	client *http.Client
}

func (d *dirverHttp) run(o *Order) {

	t := o.Task

	var rb io.Reader
	if t.Body != nil {
		rb = bytes.NewReader(t.Body)
	}

	req, err := http.NewRequestWithContext(o.ctx, t.Method, t.Url, rb)
	if err != nil {
		o.status = -1
		o.err = err
		return
	}

	o.taskTxt = req.URL.String()

	resp, err := d.client.Do(req)

	if err != nil {
		o.status = -1
		o.err = err
		return
	}

	onResponse(o, resp)
}

func onResponse(o *Order, resp *http.Response) {
	defer resp.Body.Close()

	b2, _ := io.ReadAll(resp.Body)

	o.status = resp.StatusCode
	o.msg = string(b2)

	if o.Task.Status == 0 {
		if !(o.status >= 200 && o.status < 300) {
			o.err = fmt.Errorf("Status %d", o.status)
		}
	} else if o.Task.Status != o.status {
		o.err = fmt.Errorf("Status %d != %d", o.status, o.Task.Status)
	}
}
