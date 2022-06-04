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
}

func (w *Worker) Init(id int, s *Scheduler) *Worker {
	w.Id = id
	w.s = s
	w.task = make(chan *Task)
	return w
}

func (w *Worker) exec(t *Task) (status int, msg string) {
	if w.s.cfg.Mode == MODE_CMD {
		return w.doCMD(t)
	}

	return w.doHttp(t)
}

func (w *Worker) doCMD(t *Task) (status int, msg string) {
	task := w.s.cfg.Base + " " + t.job.Name
	task = strings.TrimSpace(task)

	params := strings.Split(task, " ")
	task = params[0]
	params = params[1:]
	params = append(params, t.Params...)

	c := exec.Command(task, params...)

	timer := time.AfterFunc(w.s.cfg.TaskTimeout, func() {
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

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(body)

	return
}

func (w *Worker) log(t *Task) {

	var waitTime float64 = 0
	if t.AddTime.Unix() > 0 {
		waitTime = t.StartTime.Sub(t.AddTime).Seconds()
	}

	runTime := t.EndTime.Sub(t.StartTime).Seconds()

	d := TaskLog{
		Id:       t.Id,
		Name:     t.job.Name,
		Params:   t.Params,
		Status:   t.Status,
		WaitTime: LogSecond(waitTime),
		RunTime:  LogSecond(runTime),
		Output:   t.Msg,
	}

	msg, _ := json.Marshal(d)

	w.s.log.Printf("[Task] %s\n", msg)
}

func (w *Worker) Run() {
	for t := range w.task {
		if t == nil {
			return
		}

		t.Status, t.Msg = w.exec(t)
		w.s.complete <- t
	}
}

func (w *Worker) Close() {
	close(w.task)
}
