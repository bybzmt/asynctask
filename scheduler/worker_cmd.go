package scheduler

import (
	"errors"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

var ErrCmdStatus = errors.New("Code != 200")

type workerCli struct {
	Id int

	cmd atomic.Value
}

func (w *workerCli) Cancel() {
	_cmd := w.cmd.Load()
	if _cmd != nil {
		cmd := _cmd.(*exec.Cmd)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		w.cmd.Store(nil)
	}
}

func (w *workerCli) Run(o *order) (status int, msg string) {
	var task string

	if o.Base.CmdBase != "" {
		task = o.Base.CmdBase + " " + o.Task.Cli.Cmd
	} else {
		task = o.Task.Cli.Cmd
	}

	task = strings.TrimSpace(task)

	params := strings.Split(task, " ")
	task = params[0]
	params = params[1:]
	params = append(params, o.Task.Cli.Params...)

	c := exec.Command(task, params...)
	w.cmd.Store(c)

	defer w.cmd.Store(nil)

	timeout := o.Task.Timeout
	if o.Base.Timeout > 0 {
		if timeout < 1 || timeout > o.Base.Timeout {
			timeout = o.Base.Timeout
		}
	}
	if timeout < 1 {
		timeout = 60 * 60
	}

	timer := time.AfterFunc(time.Duration(timeout)*time.Second, w.Cancel)
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
		o.Err = err
		return
	}

	status = c.ProcessState.ExitCode()
	msg = string(out)

	if status != 0 {
		o.Err = ErrCmdStatus
	}

	return
}
