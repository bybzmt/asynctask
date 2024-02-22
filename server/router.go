package server

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Rewrite struct {
	Pattern string
	Rewrite string
	Method  string            `json:",omitempty"`
	Header  map[string]string `json:",omitempty"`
	Timeout uint              `json:",omitempty"`

	exp *regexp.Regexp
}

type Route struct {
	Pattern string
	Job     string
	Dirver  string
	Rewrite *Rewrite `json:",omitempty"`
	Note    string   `json:",omitempty"`
	Disable bool     `json:",omitempty"`

	exp *regexp.Regexp
}

type router struct {
	routes []*Route
	direct map[string]*Route
}

func (r *Rewrite) init() (err error) {
	if r.Pattern == "" {
		r.exp = nil
		return nil
	}

	r.exp, err = regexp.Compile(r.Pattern)
	if err != nil {
		return
	}

	return
}

func (r *Rewrite) rewrite(t *Task) {
	if r.Method != "" {
		t.Method = r.Method
	}

	if r.exp != nil {
		t.Url = r.exp.ReplaceAllString(t.Url, r.Rewrite)
	}

	if r.Timeout > 0 {
		if t.Timeout > r.Timeout {
			t.Timeout = r.Timeout
		}
	}

	for k, v := range r.Header {
		t.Header[k] = v
	}
}

func (r *Route) init() (err error) {
	r.exp, err = regexp.Compile(r.Pattern)
	if err != nil {
		return
	}

	if r.Rewrite != nil {
		return r.Rewrite.init()
	}

	return
}

func (r *Route) rewrite(src, dst *Task) {
	*dst = *src
	dst.Header = make(map[string]string)

	for k, v := range src.Header {
		dst.Header[k] = v
	}

	if r.Rewrite != nil {
		r.Rewrite.rewrite(dst)
	}
}

func (r *router) set(rs []*Route) error {
	r.routes = nil
	r.direct = make(map[string]*Route)

	for _, t := range rs {
		if t.Disable {
			continue
		}

		if err := t.init(); err != nil {
			return err
		}

		prefix, full := t.exp.LiteralPrefix()
		if full {
			r.direct[prefix] = t
		} else {
			r.routes = append(r.routes, t)
		}
	}

	return nil
}

func (r *router) match(name string) (string, *Route) {
	if t, ok := r.direct[name]; ok {
		return name, t
	}

	for _, r := range r.routes {
		if r.exp.MatchString(name) {
			job := r.exp.ReplaceAllString(name, r.Job)
			return job, r
		}
	}

	return "", nil
}

func (rs *router) route(t *Task) (*Order, error) {

	if t == nil {
		return nil, fmt.Errorf(`Task is nil`)
	}

	t.Url = strings.TrimSpace(t.Url)

	u := rs.urlCheck(t.Url)
	if u == nil {
		return nil, fmt.Errorf(`Task Invalid: %s`, t.Url)
	}

	job, r := rs.match(u.String())
	if r == nil {
		return nil, fmt.Errorf("Task Not Allow: %s", t.Url)
	}

	o := new(Order)
	o.Job = job
	o.Dirver = r.Dirver

	r.rewrite(t, &o.Task)

	return o, nil
}

func (r *router) urlCheck(u string) *url.URL {
	if u == "" {
		return nil
	}

	u2, err := url.Parse(u)
	if err != nil {
		return nil
	}

	if u2.Path != "" {
		u2.Path = path.Clean(u2.Path)
	}

	u2.RawQuery = ""
	u2.Fragment = ""

	return u2
}
