package scheduler

import (
	"regexp"
)

type router struct {
	TaskConfig

	exp *regexp.Regexp
}

func (r *router) init() error {
    r.TaskBase.init()

    if r.Match == "" {
        return nil
    }

	exp, err := regexp.Compile(r.Match)

	if err != nil {
		return err
	}

	r.exp = exp

	return nil
}

func (r *router) match(name string) bool {
	if !r.Used {
		return false
	}

	if r.exp == nil {
		return true
	}

	return r.exp.MatchString(name)
}
