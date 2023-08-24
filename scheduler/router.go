package scheduler

import (
	"crypto/rand"
	"encoding/binary"
	"regexp"
)

type router struct {
	RouterConfig

	id      ID
	exp     *regexp.Regexp
	weights []uint32
	total   uint32
}

func (r *router) init() {
	r.weights = make([]uint32, len(r.Weights))

	for i, w := range r.Weights {
		if w < 1 {
			w = 1
		}
		r.total += w
		r.weights[i] = r.total
	}
}

func (r *router) match(t *Task) bool {
	if r.exp == nil {
		return true
	}

	if t.Http != nil {
		if r.Mode&MODE_HTTP > 0 {
			return r.exp.MatchString(t.Name)
		}
	}
	if t.Cli != nil {
		if r.Mode&MODE_CMD > 0 {
			return r.exp.MatchString(t.Name)
		}
	}

	return false

}

func (r *router) randGroup() ID {
	if len(r.Groups) == 0 {
		return 0
	}

	if len(r.Groups) == 1 {
		return r.Groups[0]
	}

	var num uint32

	binary.Read(rand.Reader, binary.LittleEndian, &num)

	num = num % r.total

	for i, a := range r.weights {
		if num < a {
			return r.Groups[i]
		}
	}

	return 0
}
