package scheduler

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var ErrCmdStatus = errors.New("Code != 200")

type workerCli struct {
	Id   int
	l    sync.Mutex
	resp context.CancelFunc
}

func (w *workerCli) Cancel() {
	w.l.Lock()
	defer w.l.Unlock()

	if w.resp != nil {
		w.resp()
		w.resp = nil
	}
}

func (w *workerCli) Run(o *order) (status int, msg string) {
	var task string

	if o.Base.CliBase != "" {
		task = o.Base.CliBase + " " + o.Task.Cli.Cmd
	} else {
		task = o.Task.Cli.Cmd
	}

	task = strings.TrimSpace(task)

	params := strings.Split(task, " ")
	task = params[0]
	params = params[1:]
	params = append(params, o.Task.Cli.Params...)

	timeout := o.Task.Timeout
	if o.Base.Timeout > 0 {
		if timeout < 1 || timeout > o.Base.Timeout {
			timeout = o.Base.Timeout
		}
	}
	if timeout < 1 {
		timeout = 60 * 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

	c := exec.CommandContext(ctx, task, params...)
	c.Env = make([]string, 0, len(o.Base.CliEnv))

	for k, v := range o.Base.CliEnv {
		c.Env = append(c.Env, k+"="+v)
	}

	if o.Base.CliDir != "" {
		c.Dir = o.Base.CliDir
	} else {
		c.Dir = os.TempDir()
	}

	w.l.Lock()
	w.resp = cancel
	w.l.Unlock()

	defer func() {
		w.l.Lock()
		w.resp = nil
		w.l.Unlock()
	}()

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
