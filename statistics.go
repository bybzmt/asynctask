package main

import (
	"container/list"
	"time"
)

type StatRow struct {
	useTime  time.Duration
	idleTime time.Duration

	workers map[int]time.Duration
	jobs    map[int]time.Duration
}

func (s *StatRow) Init() *StatRow {
	s.workers = make(map[int]time.Duration)
	s.jobs = make(map[int]time.Duration)
	return s
}

type Statistics struct {
	LastTime   time.Time
	StatPeriod time.Duration

	times  list.List
	maxNum int
}
