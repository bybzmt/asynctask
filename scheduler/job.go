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

	tasks_head int
	tasks_tail int
	tasks      [][1024]ID

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
	p := j.tasks_tail / 1024
	f := j.tasks_tail % 1024

	l := len(j.tasks)

	if p > l-1 {
		j.tasks = append(j.tasks, [1024]ID{})
	}

	j.tasks[p][f] = taskid
	j.tasks_tail++
}

func (j *job) popTask() ID {
	if len(j.tasks) == 0 {
		return 0
	}

	if j.tasks_head == j.tasks_tail {
		return 0
	}

	p := j.tasks_head / 1024
	f := j.tasks_head % 1024

	j.tasks_head++

	if j.tasks_head == 1024 {
		j.tasks = j.tasks[1:]
		j.tasks_tail -= j.tasks_head
		j.tasks_head = 0
	}

	return j.tasks[p][f]
}

func (j *job) len() int {
	return j.tasks_tail - j.tasks_head
}
