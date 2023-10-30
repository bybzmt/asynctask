package scheduler

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

var ErrCmdStatus = errors.New("Code != 200")

type workerCli struct {
	order *order
	cmd   *exec.Cmd

	cancel context.CancelFunc
}

func (w *workerCli) init() error {

	var cmd string
	args := w.order.base.CmdArgs

	if w.order.base.CmdPath != "" {
        u, err := url.Parse(w.order.Task.Cmd)
        if err != nil {
            return err
        }

        path := strings.TrimLeft(u.Path, "/")
        if path == "" {
            return fmt.Errorf("Error Task: %s", w.order.Task.Cmd)
        }

		cmd = w.order.base.CmdPath
		args = append(args, path)
	} else {
		cmd = path.Clean(w.order.Task.Cmd)
        cmd = strings.TrimLeft(cmd, ".")
	}

	args = append(args, w.order.Task.Args...)

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
