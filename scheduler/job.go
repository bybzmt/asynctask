package scheduler

import (
	"time"
)

type job_mode int

const (
	job_mode_runnable job_mode = iota
	job_mode_block
	job_mode_idle
)

type job struct {
	priority int32
	parallel uint32

	name  string
	group string

	s *Scheduler
	g *group

	next, prev *job
	mode       job_mode

	score int

	tasks []ID

	nowNum int32
	errNum int32
	runNum int32
	oldRun int32
	oldErr int32

	//平均执行时间
	useTime  statRow
	loadStat statRow

	loadTime time.Duration
	lastTime time.Time
}

func (j *job) init(s *Scheduler, name string) *job {
	j.s = s
	j.name = name
	j.useTime.init(10)
	j.loadStat.init(j.s.statSize)
	return j
}

func (j *job) addTask(taskid ID) {
	if len(j.tasks) >= cap(j.tasks) {
		j.tasks = append(make([]ID, 0, 128), j.tasks...)
	}

	j.tasks = append(j.tasks, taskid)
}

func (j *job) popTask() ID {
	if len(j.tasks) == 0 {
		return 0
	}

	id := j.tasks[0]
	j.tasks = j.tasks[1:]

	return id
}

func (j *job) len() int {
	return len(j.tasks)
}
