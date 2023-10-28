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
		name := string(key)

		_, ok := s.jobs[name]
		if !ok {
			jt, err := s.newJob(name)
			if err != nil {
				return err
			}

			s.jobs[name] = jt
		}
	}

	return nil
}

func (s *Scheduler) getJob(t *order) (*job, error) {
	j, ok := s.jobs[t.Job]

	if !ok {
		var err error

		j, err = s.newJob(t.Job)
		if err != nil {
			return nil, err
		}

		s.jobs[t.Job] = j
	}

	return j, nil
}

func (s *Scheduler) newJob(name string) (*job, error) {

	j := new(job)
	j.s = s
	j.Name = name
    j.Parallel = s.Parallel
    j.cfgMode = 1
    c := s.db_Job_load(name)

    if c == nil {
        j.cfgMode = 2
        t := s.matchRule(name)
        if t != nil {
            c = &t.JobConfig
        }
    }

    if c != nil {
	    j.JobConfig = *c
        j.Name = name
        j.CmdEnv = copyMap(c.CmdEnv)
        j.HttpHeader = copyMap(c.HttpHeader)
    } else {
        j.cfgMode = 0
        j.GroupId = 1
    }

	j.group = s.getGroup(j.GroupId)
	j.init()

	return j, nil
}
