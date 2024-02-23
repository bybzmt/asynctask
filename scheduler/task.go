package scheduler

import ()

func (s *Scheduler) JobEmpty(job string, num int) []ID {
	s.l.Lock()
	defer s.l.Unlock()

	j, ok := s.jobs[job]
	if !ok {
		return nil
	}

	var ids []ID

	for len(ids) < num {
		id := j.popTask()
		if id == 0 {
			if j.g != nil {
				j.g.modeCheck(j)
			}

			break
		}

		ids = append(ids, id)
	}

	return ids
}

func (s *Scheduler) JobDel(job string) error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.delIdleJob(job)
}

func (s *Scheduler) TaskCancel(oid ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	for t := range s.orders {
		if t.id == oid {
			t.cancel()
			return nil
		}
	}

	return NotFound
}

func (s *Scheduler) TaskRuning() []ID {
	s.l.Lock()
	defer s.l.Unlock()

	var ids []ID

	for t := range s.orders {
		ids = append(ids, t.id)
	}

	return ids
}

func (s *Scheduler) TaskAdd(t *Task) {
	s.l.Lock()
	defer s.l.Unlock()

	j, ok := s.jobs[t.Job]

	if ok {
		if j.g == nil {
			if j.mode == job_mode_idle {
				s.idleLen--
			}

			jobRemove(j)

			j.g = s.getGroup(j.group)
			j.g.runAdd(j)
		}
	} else {
		j = s.newJob(t.Job)
		s.jobs[t.Job] = j
		j.g = s.getGroup(j.group)
		j.g.runAdd(j)
	}

	j.addTask(t.Id)
	j.g.waitNum++
	j.g.modeCheck(j)

	j.g.dispatch()
}
