package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type worker interface {
	run() (status int, msg string)
	init() error
}

// 运行的任务
type order struct {
	Id  ID
	job *job
	g   *group

	Task *Task
	Base TaskBase

	ctx    context.Context
	cancel context.CancelFunc

	Status int
	Msg    string
	Err    error

	AddTime   time.Time
	StartTime time.Time
	StatTime  time.Time
	EndTime   time.Time

	taskTxt string

	logFields map[string]any
}

func (o *order) Run() {
	o.g.s.Log.Debugln("worker run", o.Id)

	o.logFields = map[string]any{
		"tag":  "task",
		"id":   o.Task.Id,
		"name": o.Task.Name,
	}

	has := false

	if o.Base.Mode&MODE_HTTP == MODE_HTTP {
		if o.Task.Url == "" {
			o.Err = errors.New("task error")
			has = true
		} else {
			w := workerHttp{
				order: o,
			}

			if err := w.init(); err != nil {
				w.order.Err = err
				has = true
			} else {
				o.Status, o.Msg = w.run()

				o.logFields["url"] = o.taskTxt
				o.logFields["status"] = o.Status
			}
		}
	} else if o.Base.Mode&MODE_CLI == MODE_CLI {
		if o.Task.Cmd == "" {
			o.Err = errors.New("task error")
			has = true
		} else {
			w := workerCli{
				order: o,
			}

			if err := w.init(); err != nil {
				w.order.Err = err
				has = true
			} else {
				o.Status, o.Msg = w.run()

				o.logFields["cmd"] = o.taskTxt
				o.logFields["exitcode"] = o.Status
			}
		}
	} else {
		o.Err = errors.New("task error")
	}

	if has {
		if xx, err := json.Marshal(o.Task); err == nil {
			o.logFields["task"] = string(xx)
		}
	}

	o.g.complete <- o
}

func (o *order) logTask() {

	runTime := o.EndTime.Sub(o.StartTime).Seconds()

	if o.Err != nil {
		o.logFields["err"] = o.Err
	}

	o.logFields["runTime"] = fmt.Sprintf("%.2f", runTime)

	o.g.s.Log.WithFields(o.logFields).Infoln(o.Msg)
}

func logSecond() {
}
