package scheduler

import (
	"regexp"
)

type rule struct {
	Job
	exp *regexp.Regexp
}

func (r *rule) init() (err error) {
	r.exp, err = regexp.Compile(r.Pattern)
	if err != nil {
		return
	}

	return
}

type rules struct {
	routes []*rule
	direct map[string]*rule
}

func (rs *rules) set(js []*Job) error {
	rs.routes = nil
	rs.direct = make(map[string]*rule)

	for _, j := range js {
		r := &rule{Job: *j}

		if err := r.init(); err != nil {
			return err
		}

		prefix, full := r.exp.LiteralPrefix()
		if full {
			rs.direct[prefix] = r
		} else {
			rs.routes = append(rs.routes, r)
		}
	}

	return nil
}

func (rs *rules) match(name string) *rule {
	if r, ok := rs.direct[name]; ok {
		return r
	}

	for _, r := range rs.routes {
		if r.exp.MatchString(name) {
			return r
		}
	}

	return nil
}
