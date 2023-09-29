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
	JobConfig

	g *group
    task *jobTask

	next, prev *job
	mode       job_mode

	nowNum   int

    score int

	lastTime time.Time
	loadTime time.Duration
	loadStat statRow

	//任务执行所用时间
	useTimeStat statRow
}

func newJob(js *jobs, jt *jobTask) *job {
	j := new(job)
	j.JobConfig = jt.JobConfig
    j.task = jt

	j.g = js.g
	j.loadStat.init(j.g.s.statSize)
	j.useTimeStat.init(10)

	return j
}

func (j *job) countScore() {
	var x, y, z, area float64

	area = 10000

	x = float64(j.nowNum) * (area / float64(j.g.WorkerNum))

	if j.g.loadStat.getAll() > 0 {
		y = float64(j.loadStat.getAll()) / float64(j.g.loadStat.getAll()) * area
	}

    xx := j.g.s.waitNum.Load()
	if xx > 0 {
		z = area - float64(j.waitNum())/float64(xx)*area
	}

	j.score = int(x + y + z + float64(j.Priority))
}


func (j *job) popOrder() (*order, error) {
    t, err := j.task.popTask()
    if err != nil {
        return nil, err
    }

    o := new(order)
    o.Id = ID(t.Id)
    o.Task = t
    o.Base = &j.task.TaskBase
    o.AddTime = time.Unix(int64(t.AddTime), 0)

    return o, nil
}

func (j *job) hasTask() bool {
    return j.task.hasTask()
}

func (j *job) errAdd() {
    j.task.errNum.Add(1)
}

func (j *job) runAdd() {
    j.task.runNum.Add(1)
}

func (j *job) waitNum() int {
    v := j.task.waitNum.Load()
    return int(v)
}

