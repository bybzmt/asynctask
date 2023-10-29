package scheduler

type jobs struct {
	idleMax int
	idleLen int

	idle *job

	jobs map[string]*job
}

func (s *Scheduler) idleFront() *job {
	if s.idle == s.idle.next {
		return nil
	}
	return s.idle.next
}

func (s *Scheduler) idleAdd(j *job) {
	j.mode = job_mode_idle

	jobAppend(j, s.idle.prev)

	s.idleLen++

	//移除多余的idle
	for s.idleLen > s.idleMax {
		j := s.idleFront()
		if j != nil {
			s.idleLen--
			jobRemove(j)
			j.removeBucket()
		}
	}
}

func (s *Scheduler) loadJobs() error {
	buckets := db_get_buckets(s.Db, "task")

	for _, key := range buckets {
		s.jobs[key] = s.newJob(key)
	}

	return nil
}

func (s *Scheduler) newJob(name string) *job {
	j := new(job)
	j.s = s
	j.name = name

	s.ruleApply(j)

	j.init()

	return j
}
