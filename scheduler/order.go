package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

type worker interface {
	run() (status int, msg string)
	init() error
}

// 运行的任务
type Order struct {
	Id      ID `json:",omitempty"`
	Job     string
	Task    *Task      `json:",omitempty"`
	Cli     *OrderCli  `json:",omitempty"`
	Http    *OrderHttp `json:",omitempty"`
	AddTime uint

	job *job
	g   *group

	ctx    context.Context
	cancel context.CancelFunc

	status int
	msg    string
	err    error

	startTime time.Time
	statTime  time.Time
	endTime   time.Time

	taskTxt string

	logFields map[string]any
}

func (o *Order) Run() {
	o.g.s.Log.Debugln("worker run", o.Id)

	o.logFields = map[string]any{
		"id": o.Id,
	}

	has := false

	if o.Http != nil {
		o.status, o.msg = runHttp(o)

		o.logFields["url"] = o.taskTxt
		o.logFields["status"] = o.status
	} else if o.Cli != nil {
		o.status, o.msg = runCli(o)

		o.logFields["cmd"] = o.taskTxt
		o.logFields["exit"] = o.status
	} else {
		o.err = TaskError
		has = true
	}

	if has {
		if xx, err := json.Marshal(o.Task); err == nil {
			o.logFields["task"] = string(xx)
		}
	}

	o.logTask()

	o.g.s.complete <- o
}

func (o *Order) logTask() {
	o.endTime = time.Now()

	runTime := o.endTime.Sub(o.startTime).Seconds()

	o.logFields["job"] = o.job.name

	if o.err != nil {
		o.logFields["err"] = o.err

		if o.Task.Retry > 0 {
			o.logFields["retry"] = o.Task.Retry
		}
	}

	o.logFields["cost"] = logCost(runTime)

	o.g.s.Log.WithFields(o.logFields).Infoln(o.msg)
}

func logCost(ts float64) string {
	if ts > 10 {
		return fmt.Sprintf("%ds", int(ts))
	}

	return fmt.Sprintf("%.2fs", ts)
}

func runCli(o *Order) (status int, msg string) {

	t := o.Cli

	timeout := o.Task.Timeout
	if timeout < 1 {
		timeout = 60 * 60
	}

	c := exec.CommandContext(o.ctx, t.Path, t.Args...)
	c.Env = t.Env

	o.taskTxt = c.String()

	out, err := c.CombinedOutput()
	if err != nil {
		if c.ProcessState != nil {
			status = c.ProcessState.ExitCode()
		} else {
			status = -1
		}

		if len(out) == 0 {
			msg = err.Error()
		} else {
			msg = string(out)
		}
		o.err = err
		return
	}

	status = c.ProcessState.ExitCode()
	msg = string(out)

	if status != o.Task.Code {
		o.err = fmt.Errorf("Code != %d", o.Task.Code)
	}

	return
}

func runHttp(o *Order) (status int, msg string) {

	t := o.Http

	timeout := o.Task.Timeout

	if timeout < 1 {
		timeout = 60
	}

	var rb io.Reader
	if t.Body != nil {
		rb = bytes.NewReader(t.Body)
	}

	req, err := http.NewRequestWithContext(o.ctx, t.Method, t.Url, rb)
	if err != nil {
		status = -1
		o.err = err
		return
	}

	o.taskTxt = req.URL.String()

	resp, err := o.job.s.Client.Do(req)
	if err != nil {
		status = -1
		o.err = err
		return
	}

	defer resp.Body.Close()

	b2, _ := io.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(b2)

	if o.Task.Code == 0 {
		if !(status >= 200 && status < 300) {
			o.err = fmt.Errorf("Status %d", status)
		}
	} else if o.Task.Code != status {
		o.err = fmt.Errorf("Status %d != %d", status, o.Task.Code)
	}

	return
}
