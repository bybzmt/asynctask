package scheduler

import (
	"regexp"
)

type router struct {
	RouterConfig

	id      ID
	exp     *regexp.Regexp
}

func (r *router) init() {
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

