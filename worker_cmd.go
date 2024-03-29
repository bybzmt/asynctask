package main

import (
	"errors"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

var ErrCmdStatus = errors.New("Code != 200")

type WorkerCmd struct {
	Id int

	cmd atomic.Value

	task chan *Task
	s    *Scheduler
}

func (w *WorkerCmd) Init(id int, s *Scheduler) *WorkerCmd {
	w.Id = id
	w.s = s
	w.task = make(chan *Task)
	return w
}

func (w *WorkerCmd) Exec(t *Task) {
	w.task <- t
}

func (w *WorkerCmd) Cancel() {
	_cmd := w.cmd.Load()
	if _cmd != nil {
		cmd := _cmd.(*exec.Cmd)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
}

func (w *WorkerCmd) Run() {
	for t := range w.task {
		if t == nil {
			return
		}

		t.Status, t.Msg = w.doCMD(t)

		w.s.complete <- t
	}
}

func (w *WorkerCmd) Close() {
	close(w.task)
}

func (w *WorkerCmd) doCMD(t *Task) (status int, msg string) {
	task := w.s.cfg.Base + " " + t.job.Name
	task = strings.TrimSpace(task)

	params := strings.Split(task, " ")
	task = params[0]
	params = params[1:]
	params = append(params, t.Params...)

	c := exec.Command(task, params...)
	w.cmd.Store(c)

	timer := time.AfterFunc(w.s.cfg.TaskTimeout, w.Cancel)
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
		t.Err = err
		return
	}

	status = c.ProcessState.ExitCode()
	msg = string(out)

	if status != 0 {
		t.Err = ErrCmdStatus
	}

	return
}
