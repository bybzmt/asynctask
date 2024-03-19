package server

import (
	rbtree "github.com/sakeven/RbTree"
	"time"
)

type timePoint struct {
	point int64
	tasks []ID
}

type timer struct {
	tp  *rbtree.Tree[int64, *timePoint]
	num int
}

func (t *timer) init() {
	t.tp = rbtree.NewTree[int64, *timePoint]()
}

func (t *timer) pop(unixSec int64) []ID {

	if t.tp.Size() < 1 {
		return nil
	}

	min := t.tp.Iterator()

	if unixSec < min.Key {
		return nil
	}

	defer t.tp.Delete(min.Key)

	ids := min.Value.tasks

	t.num -= len(ids)

	return ids
}

func (t *timer) push(unixSec int64, taskid ID) {
	t.num++

	p := t.tp.Find(unixSec)

	if p == nil {
		p = &timePoint{
			point: unixSec,
			tasks: []ID{taskid},
		}
		t.tp.Insert(unixSec, p)
	} else {
		p.tasks = append(p.tasks, taskid)
	}
}

func (t *timer) Len() int {
	return t.num
}

func (t *timer) TopN(num int) []ID {
	if t.tp.Size() < 1 {
		return nil
	}

	n := t.tp.Iterator()

	var out []ID

	for n != nil {
		out = append(out, n.Value.tasks...)

		if len(out) >= num {
			break
		}

		n = n.Next()
	}

	return out[:num]
}

func (s *Server) TimerTopN(num int) []*Order {
	s.l.Lock()
	defer s.l.Unlock()

	ids := s.timer.TopN(num)

	var out []*Order

	for _, id := range ids {
		o := s.store_order_get(id)

		if o != nil {
			out = append(out, o)
		} else {
			s.log.Errorln("TimerTopN task NotFound", id)
		}
	}

	return out
}

func (s *Server) checkTimer(now time.Time) {
	ts := now.Unix()
	for {
		s.l.Lock()
		ids := s.timer.pop(ts)
		s.l.Unlock()

		if ids == nil {
			break
		}

		for _, id := range ids {
			o := s.store_order_get(id)

			if o != nil {
				s.s.TaskAdd(&task{
					Id:  o.Id,
					Job: o.Job,
				})
			} else {
				s.log.Errorln("checkTimer task NotFound", id)
			}
		}
	}
}
