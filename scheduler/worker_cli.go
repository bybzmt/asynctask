package scheduler

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"
)

var ErrCmdStatus = errors.New("Code != 200")

type workerCli struct {
    order *order
    cmd *exec.Cmd

    cancel context.CancelFunc
}


func (w *workerCli) init() error {

	var cmd string
    args := []string{}

	if w.order.Base.CmdBase != "" {
        cmd = w.order.Base.CmdBase
        args = append(args, w.order.Task.Cmd)
	} else {
		cmd = w.order.Task.Cmd
	}

    args = append(args, w.order.Task.Args...)

	timeout := w.order.Task.Timeout
	if w.order.Base.Timeout > 0 {
		if timeout < 1 || timeout > w.order.Base.Timeout {
			timeout = w.order.Base.Timeout
		}
	}
	if timeout < 1 {
		timeout = 60 * 60
	}

	ctx, cancel := context.WithTimeout(w.order.ctx, time.Duration(timeout)*time.Second)
    w.cancel = cancel

	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = make([]string, 0, len(w.order.Base.CmdEnv))

	for k, v := range w.order.Base.CmdEnv {
		c.Env = append(c.Env, k+"="+v)
	}

	if w.order.Base.CmdDir != "" {
		c.Dir = w.order.Base.CmdDir
	} else {
		c.Dir = os.TempDir()
	}

    w.cmd = c

    return nil
}

func (w *workerCli) run() (status int, msg string) {
	if err := w.init(); err != nil {
		status = -1
		w.order.Err = err
		return
	}

    w.order.taskTxt = w.cmd.String()

    defer w.cancel()

	out, err := w.cmd.CombinedOutput()
	if err != nil {
		if w.cmd.ProcessState != nil {
			status = w.cmd.ProcessState.ExitCode()
		} else {
			status = -1
		}

		if len(out) == 0 {
			msg = err.Error()
		} else {
			msg = string(out)
		}
		w.order.Err = err
		return
	}

	status = w.cmd.ProcessState.ExitCode()
	msg = string(out)

	if status != w.order.Task.Code {
		w.order.Err = ErrCmdStatus
	}

	return
}
