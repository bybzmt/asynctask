package scheduler

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
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
    args := w.order.Task.Args

    str := strings.TrimSpace(w.order.Task.Cmd)

	u, err := url.Parse(str)
	if err != nil {
		return err
	}

	path := strings.TrimLeft(u.Path, "/")

	if w.order.base.CmdBase != "" {
        cmd = w.order.base.CmdBase
        args = append(args, path)
	} else {
		cmd = path
	}

	timeout := w.order.Task.Timeout
	if w.order.base.Timeout > 0 {
		if timeout < 1 || timeout > w.order.base.Timeout {
			timeout = w.order.base.Timeout
		}
	}
	if timeout < 1 {
		timeout = 60 * 60
	}

	ctx, cancel := context.WithTimeout(w.order.ctx, time.Duration(timeout)*time.Second)
    w.cancel = cancel

	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = make([]string, 0, len(w.order.base.CmdEnv))

	for k, v := range w.order.base.CmdEnv {
		c.Env = append(c.Env, k+"="+v)
	}

	if w.order.base.CmdDir != "" {
		c.Dir = w.order.base.CmdDir
	} else {
		c.Dir = os.TempDir()
	}

    w.cmd = c

    return nil
}

func (w *workerCli) run() (status int, msg string) {
	if err := w.init(); err != nil {
		status = -1
		w.order.err = err
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
		w.order.err = err
		return
	}

	status = w.cmd.ProcessState.ExitCode()
	msg = string(out)

	if status != w.order.Task.Code {
		w.order.err = ErrCmdStatus
	}

	return
}
