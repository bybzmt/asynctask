package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Worker struct {
	Id int

	run bool

	task chan *Task
	s    *Scheduler

	LastTime time.Time
}

func (w *Worker) Init(id int, s *Scheduler) *Worker {
	w.Id = id
	w.s = s
	w.task = make(chan *Task)
	return w
}

func (w *Worker) exec(t *Task) (status int, msg string) {
	if w.s.e.Mode == MODE_CMD {
		return w.doCMD(t)
	}

	return w.doHttp(t)
}

func (w *Worker) doCMD(t *Task) (status int, msg string) {
	task := w.s.e.Base + " " + t.job.Name
	task = strings.TrimSpace(task)

	params := strings.Split(task, " ")
	task = params[0]
	params = params[1:]
	params = append(params, t.Params...)

	c := exec.Command(task, params...)

	timer := time.AfterFunc(w.s.e.Timeout, func() {
		if c.Process != nil {
			c.Process.Kill()
		}
	})
	defer timer.Stop()

	out, err := c.CombinedOutput()
	if err != nil {
		if c.ProcessState != nil {
			status = c.ProcessState.ExitCode()
		} else {
			status = 1
		}

		if len(out) == 0 {
			msg = err.Error()
		} else {
			msg = string(out)
		}
		return
	}

	status = c.ProcessState.ExitCode()
	msg = string(out)
	return
}

func (w *Worker) doHttp(t *Task) (status int, msg string) {
	var url string

	url = w.s.e.Base + t.job.Name

	var resp *http.Response
	var err error

	if strings.IndexByte(url, '?') == -1 {
		url = url + "?" + strings.Join(t.Params, "&")
	} else {
		url = url + "&" + strings.Join(t.Params, "&")
	}

	resp, err = w.s.e.Client.Get(url)

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

type TaskLog struct {
	Params []string
	Output string
}

func (w *Worker) log(t *Task) {

	d := TaskLog{
		Params: t.Params,
		Output: t.Msg,
	}

	msg, _ := json.Marshal(d)

	w.s.e.Log.Printf(
		"[Task] id:%d %s %d %0.3fs %0.3fs %s\n",
		t.Id,
		t.job.Name,
		t.Status,
		t.StartTime.Sub(t.AddTime).Seconds(),
		t.EndTime.Sub(t.StartTime).Seconds(),
		msg,
	)
}

func (w *Worker) Run() {
	for t := range w.task {
		t.StartTime = time.Now()
		t.Status, t.Msg = w.exec(t)
		t.EndTime = time.Now()

		w.log(t)

		w.s.complete <- t
	}
}

func (w *Worker) Close() {
	close(w.task)
}
