package scheduler

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

func (s *Scheduler) SetRoutes(pattern []string) error {
	s.router.l.Lock()
	defer s.router.l.Unlock()

	if len(pattern) == 0 {
		return errors.New("pattern empty")
	}

	var exps []*regexp.Regexp

	for _, p := range pattern {
		if p == "" {
			return errors.New("pattern empty")
		}

		exp, err := regexp.Compile(p)
		if err != nil {
			return err
		}

		exps = append(exps, exp)
	}

	s.router.routes = pattern
	s.router.exps = exps

	return nil
}

func (s *Scheduler) Routes() []string {
	s.router.l.Lock()
	defer s.router.l.Unlock()

	return s.router.routes
}

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

func (s *Scheduler) JobDelIdle(name string) error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.delIdleJob(name)
}

func (s *Scheduler) JobEmpty(jname string) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

	return jt.delAllTask()
}

func (s *Scheduler) DelTask(jname string, tid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	jt, ok := s.jobs[jname]
	if !ok {
		return NotFound
	}

	return jt.delTask(tid)
}

func (s *Scheduler) taskCheck(t *Task) (string, error) {
	t.Cmd = strings.TrimSpace(t.Cmd)
	t.Url = strings.TrimSpace(t.Url)

	if (t.Url == "" && t.Cmd == "") || (t.Url != "" && t.Cmd != "") {
		return "", TaskError
	}

	var str string
	var scheme string

	if t.Cmd != "" {
		t.Cmd = path.Clean(t.Cmd)

		if t.Cmd == "/" || t.Cmd == "." {
			return "", fmt.Errorf("Cmd Invalid: %s", t.Cmd)
		}

		scheme = "cli"
		str = t.Cmd
	} else {
		scheme = "http"
		str = t.Url
	}

	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}

	if u.Scheme == "" {
		u.Scheme = scheme
	}

	u.RawQuery = ""
	u.Fragment = ""

	path := u.String()
	path = strings.Trim(path, "/")

	return path, nil
}

func (s *Scheduler) TaskAdd(t Task) error {
	path, err := s.taskCheck(&t)
	if err != nil {
		return err
	}

	job := s.router.Route(path)

	if job == "" {
		return errors.New("Task Not Allow")
	}

	s.l.Lock()
	defer s.l.Unlock()

	o := new(order)
	o.Task = t
	o.Job = job

	s.TaskNextId++

	o.Id = s.TaskNextId
	o.AddTime = uint(s.now.Unix())

	if o.Task.Timer > o.AddTime {
		return s.timerAddTask(o)
	}

	return s.addTask(o)
}

func (s *Scheduler) Running() bool {
	s.l.Lock()
	defer s.l.Unlock()

	return s.running
}
