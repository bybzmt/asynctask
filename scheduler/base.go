package scheduler

import (
	"errors"
)

var empty = errors.New("Empty")
var NotFound = errors.New("NotFound")
var JobBusy = errors.New("JobBusy")
var DirverError = errors.New("Dirver is nil")

func jobAppend(j, at *job) {
	at.next.prev = j
	j.next = at.next
	j.prev = at
	at.next = j
}

func jobRemove(j *job) {
	j.prev.next = j.next
	j.next.prev = j.prev
	j.next = nil
	j.prev = nil
}

func jobMoveBefore(j, x *job) {
	if j == x {
		return
	}

	jobRemove(j)
	jobAppend(j, x.prev)
}

type nullLogger struct{}

func (nullLogger) Print(...interface{})          {}
func (nullLogger) Printf(string, ...interface{}) {}
func (nullLogger) Println(...interface{})        {}
