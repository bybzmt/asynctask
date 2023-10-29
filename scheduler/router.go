package scheduler

import (
	"regexp"
	"sync"
)

type router struct {
	l sync.Mutex

	routes []string
	exps   []*regexp.Regexp
}

func (r *router) init() {
	for _, ex := range r.routes {
		exp, err := regexp.Compile(ex)

		if err == nil {
			r.exps = append(r.exps, exp)
		}
	}

	r.initDefault()
}

func (r *router) initDefault() {
	if len(r.exps) == 0 {
		p := "^https?://"
		r.routes = append(r.routes, p)
		exp, _ := regexp.Compile(p)
		r.exps = append(r.exps, exp)
	}
}

func (r *router) Route(job string) string {
	r.l.Lock()
	defer r.l.Unlock()

	for _, exp := range r.exps {
		p := exp.FindStringSubmatch(job)

		l := len(p)

		if l == 1 {
			return job
		} else if l > 1 {
			return p[1]
		}
	}

	return ""
}

func (s *Scheduler) db_router_save(routes []string) error {
	return db_put(s.Db, routes, "config", "router.cfg")
}

func (s *Scheduler) db_router_load() (out []string, err error) {
	err = db_fetch(s.Db, &out, "config", "router.cfg")
	return
}
