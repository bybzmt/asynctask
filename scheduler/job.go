package scheduler

import (
	"sync/atomic"
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

    id ID
	Name     string

	g *group
    task *jobTask

	next, prev *job
	mode       job_mode

	NowNum   int

    Score int

	LastTime time.Time
	LoadTime time.Duration
	LoadStat StatRow

	//任务执行所用时间
	UseTimeStat StatRow
}

func newJob(js *jobs, jtask *jobTask) *job {
	var cfg JobConfig

	j := new(job)
	j.JobConfig = cfg
    j.task = jtask
	j.init(jtask.name, js.g)

	return j
}

func (j *job) init(name string, g *group) *job {
	j.Name = name
	j.g = g
	j.LoadStat.Init(j.g.s.StatSize)
	j.UseTimeStat.Init(10)
	j.Parallel = j.g.Parallel
	return j
}

func (j *job) countScore() {
	var x, y, z, area float64

	area = 10000

	x = float64(j.NowNum) * (area / float64(j.g.WorkerNum))

	if j.g.LoadStat.GetAll() > 0 {
		y = float64(j.LoadStat.GetAll()) / float64(j.g.LoadStat.GetAll()) * area
	}

	if j.g.WaitNum > 0 {
		z = area - float64(j.waitNum())/float64(j.g.WaitNum)*area
	}

	j.Score = int(x + y + z + float64(j.Priority))
}


func (j *job) popOrder() (*order, error) {
    t, err := j.task.popTask()
    if err != nil {
        return nil, err
    }

    o := new(order)
    o.Id = ID(t.Id)
    o.Task = t
    o.Base = j.task.base
    o.AddTime = time.Unix(int64(t.AddTime), 0)

    return o, nil
}


func (j *job) errAdd() {
    atomic.AddInt32(&j.task.errNum, 1)
}

func (j *job) runAdd() {
    atomic.AddInt32(&j.task.runNum, 1)
}

func (j *job) runNum() int {
    v := atomic.LoadInt32(&j.task.runNum)
    return int(v)
}

func (j *job) errNum() int {
    v := atomic.LoadInt32(&j.task.errNum)
    return int(v)
}

func (j *job) waitNum() int {
    v := atomic.LoadInt32(&j.task.waitNum)
    return int(v)
}

func (j *job) oldNum() int {
    v := atomic.LoadInt32(&j.task.oldNum)
    return int(v)
}

func (j *job) dayChange() {
    n1 := atomic.SwapInt32(&j.task.runNum, 0)
    atomic.StoreInt32(&j.task.errNum, 0)
    atomic.StoreInt32(&j.task.oldNum, n1)
}
