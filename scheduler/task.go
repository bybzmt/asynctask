package scheduler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
)

func (s *Scheduler) TaskCancel(oid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	for t := range s.orders {
		if t.Id == oid {
			t.cancel()
			return nil
		}
	}

	return NotFound
}

func (s *Scheduler) TaskDelIdle(name string) error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.delIdleJob(name)
}

func (s *Scheduler) TaskEmpty(name string) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[name]
	if !ok {
		return NotFound
	}

	return jt.delAllTask()
}

func (s *Scheduler) TaskDel(name string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	j, ok := s.jobs[name]
	if !ok {
		return NotFound
	}

	return j.delTask(tid)
}

func (s *Scheduler) TaskCheck(t Task) (*Order, error) {
	return s.taskCheck(&t)
}

func (s *Scheduler) taskCheck(t *Task) (*Order, error) {
	s.tl.Lock()
	defer s.tl.Unlock()

	if t == nil {
		return nil, fmt.Errorf(`Task is nil`)
	}

	u := s.taskNameCheck(t)
	if u == nil {
		return nil, fmt.Errorf(`Task Name Invalid: %s`, t.Name)
	}

	name, r := s.matchTask(u.String())
	if name == "" {
		return nil, fmt.Errorf("Task Not Allow: %s", t.Name)
	}

	o := new(Order)
	o.Task = t
	o.Job = name

	if u.Scheme == "http" {
		o.Http = &OrderHttp{
			Method: t.Method,
			Header: url.Values{},
		}

		for k, v := range t.Header {
			o.Http.Header.Set(k, v)
		}

		for k, v := range r.TaskBase.HttpHeader {
			o.Http.Header.Set(k, v)
		}

		if r.exp2 != nil {
			o.Http.Url = r.exp2.ReplaceAllString(t.Name, r.RewriteRepl)
		} else if r.RewriteRepl != "" {
			o.Http.Url = r.RewriteRepl
		} else {
			o.Http.Url = t.Name
		}

		ty := o.Http.Header.Has("Content-Type")

		if t.Body == nil && t.Args == nil {
			if o.Http.Method == "" {
				o.Http.Method = "GET"
			}
		} else {
			if o.Http.Method == "" {
				o.Http.Method = "POST"
			}

			if t.Body != nil {
				if !ty {
					o.Http.Header.Set("Content-Type", "application/json")
				}

				o.Http.Body = t.Body
			} else {
				var str string
				var kv map[string]string

				if err := json.Unmarshal(t.Args, &str); err == nil {
					if !ty {
						o.Http.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					}

					o.Http.Body = []byte(str)
				} else if err := json.Unmarshal(t.Args, &kv); err == nil {
					if !ty {
						o.Http.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					}

					t := url.Values{}
					for k, v := range kv {
						t.Set(k, v)
					}

					o.Http.Body = []byte(t.Encode())
				} else {
					if !ty {
						o.Http.Header.Set("Content-Type", "application/json")
					}

					o.Http.Body = t.Args
				}
			}

			if r.Timeout == 0 {
				r.Timeout = 60
			}
		}
	} else if u.Scheme == "cli" {
		o.Cli = &OrderCli{
			Args: r.TaskBase.CmdArgs,
			Dir:  r.TaskBase.CmdDir,
			Env:  []string{},
		}

		for k, v := range r.TaskBase.CmdEnv {
			o.Cli.Env = append(o.Cli.Env, k+"="+v)
		}

		path := u.Path

		if r.exp2 != nil {
			path = r.exp2.ReplaceAllString(u.String(), r.RewriteRepl)
		} else if r.RewriteRepl != "" {
			path = r.RewriteRepl
		}

		if r.TaskBase.CmdPath == "" {
			o.Cli.Path = path
		} else {
			o.Cli.Path = r.TaskBase.CmdPath
			o.Cli.Args = append(o.Cli.Args, path)
		}

		if o.Cli.Dir == "" {
			o.Cli.Dir = os.TempDir()
		}

		var args []string

		if t.Args != nil {
			err := json.Unmarshal(t.Args, &args)
			if err != nil {
				return nil, fmt.Errorf("Task Args need type []string")
			}
		}

		o.Cli.Args = append(o.Cli.Args, args...)

		if r.Timeout == 0 {
			r.Timeout = 60 * 60
		}
	} else {
		return nil, fmt.Errorf("Task Not Allow Scheme: %s", u.Scheme)
	}

	if t.Timeout == 0 || t.Timeout > r.Timeout {
		t.Timeout = r.Timeout
	}

	return o, nil
}

func (s *Scheduler) taskNameCheck(t *Task) *url.URL {
	t.Name = strings.TrimSpace(t.Name)

	if t.Name == "" {
		return nil
	}

	u, err := url.Parse(t.Name)
	if err != nil {
		return nil
	}

	if u.Path != "" {
		u.Path = path.Clean(u.Path)
	}

	u.RawQuery = ""
	u.Fragment = ""

	return u
}

func (s *Scheduler) TaskAdd(t Task) error {
	o, err := s.taskCheck(&t)
	if err != nil {
		return err
	}

	s.l.Lock()
	defer s.l.Unlock()

	s.TaskNextId++

	o.Id = s.TaskNextId
	o.AddTime = uint(s.now.Unix())

	if o.Task.Timer > o.AddTime {
		return s.timerAddTask(o)
	}

	return s.addTask(o)
}
