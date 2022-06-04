package main

import (
	"io/ioutil"
	"net/http"
	"strings"
)

type WorkerHttp struct {
	Id int

	resp *http.Response

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
	if w.resp != nil {
		w.resp.Body.Close()
		w.resp = nil
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

	resp, err = w.s.cfg.Client.Get(url)

	if err != nil {
		status = -1
		msg = err.Error()
		return
	}

	w.resp = resp

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(body)

	return
}
