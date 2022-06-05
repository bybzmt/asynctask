package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
)

type WorkerHttp struct {
	Id int

	resp atomic.Value

	task chan *Task
	s    *Scheduler
}

func (w *WorkerHttp) Init(id int, s *Scheduler) *WorkerHttp {
	w.Id = id
	w.s = s
	w.task = make(chan *Task)
	return w
}

func (w *WorkerHttp) Exec(t *Task) {
	w.task <- t
}

func (w *WorkerHttp) Cancel() {
	_resp := w.resp.Load()

	if _resp != nil {
		fn := _resp.(context.CancelFunc)
		if fn != nil {
			fn()
		}
	}
}

func (w *WorkerHttp) Run() {
	for t := range w.task {
		if t == nil {
			return
		}

		t.Status, t.Msg = w.doHttp(t)

		w.s.complete <- t
	}
}

func (w *WorkerHttp) Close() {
	close(w.task)
}

func (w *WorkerHttp) doHttp(t *Task) (status int, msg string) {
	var url string

	url = w.s.cfg.Base + t.job.Name

	var resp *http.Response
	var err error

	if strings.IndexByte(url, '?') == -1 {
		url = url + "?" + strings.Join(t.Params, "&")
	} else {
		url = url + "&" + strings.Join(t.Params, "&")
	}

	ctx, cancel := context.WithCancel(context.Background())

	w.resp.Store(cancel)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		status = -1
		msg = err.Error()
		return
	}

	resp, err = w.s.cfg.Client.Do(req)
	if err != nil {
		status = -1
		msg = err.Error()
		return
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(body)

	return
}
