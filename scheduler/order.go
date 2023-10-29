package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"
)

type worker interface {
	run() (status int, msg string)
	init() error
}

// 运行的任务
type order struct {
	Id      ID
	AddTime uint
	Job     string
	Task    Task
	base    TaskBase

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

func (o *order) Run() {
	o.g.s.Log.Debugln("worker run", o.Id)

	o.logFields = map[string]any{
		"tag": "task",
		"id":  o.Id,
	}

	mode := MODE_HTTP

	{
		u, err := url.Parse(o.Job)
		if err == nil {
			if u.Scheme == "cli" {
				mode = MODE_CLI
			}
		}
	}

	has := false

	if mode == MODE_HTTP {
		if o.Task.Url == "" {
			o.err = errors.New("task error")
			has = true
		} else {
			w := workerHttp{
				order: o,
			}

			if err := w.init(); err != nil {
				w.order.err = err
				has = true
			} else {
				o.status, o.msg = w.run()

				o.logFields["url"] = o.taskTxt
				o.logFields["status"] = o.status

				// if o.err != nil {
				//     t := bytes.Buffer{}
				//     w.req.Write(&t)
				//     o.logFields["req"] = string(t.Bytes())
				// }
			}
		}
	} else if mode == MODE_CLI {
		if o.Task.Cmd == "" {
			o.err = errors.New("task error")
			has = true
		} else {
			w := workerCli{
				order: o,
			}

			if err := w.init(); err != nil {
				w.order.err = err
				has = true
			} else {
				o.status, o.msg = w.run()

				o.logFields["cmd"] = o.taskTxt
				o.logFields["exit"] = o.status
			}
		}
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

func (o *order) logTask() {
	o.endTime = time.Now()

	runTime := o.endTime.Sub(o.startTime).Seconds()

	if o.err != nil {
		o.logFields["job"] = o.job.name
		o.logFields["err"] = o.err

		if o.Task.Retry > 0 {
			o.logFields["retry"] = o.Task.Retry
		}
	}

	o.logFields["cost"] = fmt.Sprintf("%.2fs", runTime)

	o.g.s.Log.WithFields(o.logFields).Infoln(o.msg)
}

func logSecond() {
}
