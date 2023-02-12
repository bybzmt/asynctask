package main

import (
	"time"

	"github.com/liyue201/gostl/ds/rbtree"
	"github.com/liyue201/gostl/utils/comparator"
)

func (s *Scheduler) initTimer() {
	s.timing = rbtree.New[uint, *Order](comparator.UintComparator)
}

func (s *Scheduler) runTimer() {
	t := time.Tick(time.Second)

	for {
		now := <-t

		for !s.running {
			return
		}

		s.checkTimer(now)
	}
}

func (s *Scheduler) checkTimer(now time.Time) {
	s.timinglock.Lock()
	defer s.timinglock.Unlock()

	n := s.timing.First()
	if n == nil {
		return
	}

	if now.Unix() > int64(n.Key()) {
		o := n.Value()
		s.timing.Delete(n)
		s.order <- o
	}
}

func (s *Scheduler) eachTimer(fn func(*Order)) {
	s.timinglock.Lock()
	defer s.timinglock.Unlock()

	n := s.timing.First()

	for n != nil {
		o := n.Value()
		fn(o)
		n = n.Next()
	}
}
