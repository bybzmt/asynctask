package server

import "time"

func (s *Server) TaskEmpty(job string) error {
	s.l.Lock()
	defer s.l.Unlock()

	for {
		ids := s.s.JobEmpty(job, 1000)

		if len(ids) == 0 {
			break
		}

		for _, id := range ids {
			s.store_order_del(id)
		}
	}

	return nil
}

func (s *Server) TaskCancel(id ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.s.TaskCancel(id)
}

func (s *Server) JobDel(job string) error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.s.JobDel(job)
}

func (s *Server) TaskDel(id ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	s.store_order_del(id)
	return nil
}

func (s *Server) TaskCheck(t *Task) (*Order, error) {
	s.l.Lock()
	defer s.l.Unlock()

	return s.router.route(t)
}

func (s *Server) TaskAdd(t *Task) error {
	s.l.Lock()
	defer s.l.Unlock()

	o, err := s.router.route(t)
	if err != nil {
		return err
	}

	o.AddTime = time.Now().Unix()

	if err := s.store_order_add(o); err != nil {
		return err
	}

	if o.Task.RunAt > o.AddTime {
		s.timer.push(o.Task.RunAt, o.Id)
		return nil
	}

	t2 := task{
		Id:  o.Id,
		Job: o.Job,
	}

	s.s.TaskAdd(&t2)

	return nil
}

func (s *Server) TaskRuning() []*Order {
	s.l.Lock()
	defer s.l.Unlock()

	ids := s.s.TaskRuning()

	var out []*Order

	for _, id := range ids {
		o := s.store_order_get(ID(id))

		if o != nil {
			out = append(out, o)
		}
	}

	return out
}
