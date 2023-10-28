package scheduler

import (
	"context"
	"time"
)

type group struct {
	GroupConfig

	s *Scheduler

	block *job
	run   *job

	ctx    context.Context
	cancel context.CancelFunc

	tick    chan time.Time

	nowNum  int
	runNum  int
	errNum  int
	oldRun  int
	oldErr  int
	waitNum int

	loadTime time.Duration
	loadStat statRow
}

func (g *group) init() {
	g.tick = make(chan time.Time)

	g.block = &job{}
	g.block.next = g.block
	g.block.prev = g.block

	g.run = &job{}
	g.run.next = g.run
	g.run.prev = g.run

	g.loadStat.init(g.s.statSize)

	g.ctx, g.cancel = context.WithCancel(context.Background())
}

func (g *group) dispatch() bool {
	if g.nowNum >= int(g.WorkerNum) {
		return false
	}

	t, err := g.GetOrder()
	if err != nil {
		if err == Empty {
			if g.nowNum == 0 {
				g.waitNum = 0
			}
			g.s.Log.Debugln("Group", g.Id, "Empty")
			return false
		}
		g.s.Log.Warnln("GetTask Error", err)

		return false
	}

	t.startTime = g.s.now
	t.statTime = g.s.now

	t.ctx, t.cancel = context.WithCancel(g.ctx)

	g.s.orders[t] = struct{}{}

	//总状态
	g.nowNum++

	go t.Run()

	return g.nowNum+1 < int(g.WorkerNum)
}

func (g *group) end(t *order) {
	loadTime := t.endTime.Sub(t.statTime)
	useTime := t.endTime.Sub(t.startTime)

	if t.err != nil {
		g.errNum += 1
		t.job.errNum += 1
	} else {
		if t.Task.Retry > 0 {
			t.Task.Retry--

			var sec uint = 60

			if t.Task.RetrySec > 0 {
				sec = t.Task.RetrySec
			}

			t.Task.Timer = uint(g.s.now.Unix()) + sec

			g.s.timerAddTask(t)
		}
	}

	t.job.end(g.s.now, loadTime, useTime)

	g.modeCheck(t.job)

	g.runNum++
	g.nowNum--
	g.loadTime += loadTime

	delete(g.s.orders, t)
}

func (g *group) statMaintain() {
	g.loadStat.push(int64(g.loadTime))
	g.loadTime = 0

	for j := g.run.next; j != g.run; j = j.next {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}

	for j := g.block.next; j != g.block; j = j.next {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}
}

func (s *Scheduler) statMaintain(now time.Time) {
	for t := range s.orders {
		us := now.Sub(t.statTime)
		t.statTime = now

		t.g.loadTime += us
		t.job.loadTime += us
	}

	for j := s.idle.next; j != s.idle; j = j.next {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}

	for _, g := range s.groups {
		g.statMaintain()
	}
}

func (g *group) dayChange() {
	g.oldRun = g.runNum
	g.oldErr = g.errNum
	g.runNum = 0
	g.errNum = 0
}


func (g *group) addJob(j *job) {
	g.runAdd(j)

	g.modeCheck(j)
}


func (g *group) modeCheck(j *job) {
	if j.next == nil || j.prev == nil {
		j.s.Log.Warning("modeCheck nil")
		return
	}

	if j.nowNum >= int32(j.Parallel) || (j.waitNum < 1 && j.nowNum > 0) {
		if j.mode != job_mode_block {
			jobRemove(j)
			g.blockAdd(j)
		}
	} else if j.waitNum < 1 {
		if j.mode != job_mode_idle {
			jobRemove(j)
			j.s.idleAdd(j)
		}
	} else {
		j.countScore()

		if j.mode != job_mode_runnable {
			jobRemove(j)
			g.runAdd(j)
		}

		g.priority(j)
	}
}

func (g *group) GetOrder() (*order, error) {
	if g.run == g.run.next {
		return nil, Empty
	}

    j := g.run.next

	o, err := j.popTask()

    g.modeCheck(j)

	if err != nil {
		if err == Empty {
			j.s.Log.Warnln("Job PopOrder Empty")

			j.waitNum = 0
		}
		return nil, err
	}

	o.g = j.group
	o.job = j

	return o, nil
}

func (g *group) front() *job {
	if g.run == g.run.next {
		return nil
	}
	return g.run.next
}

func (g *group) blockAdd(j *job) {
	j.mode = job_mode_block
	jobAppend(j, g.block.prev)
}

func (g *group) runAdd(j *job) {
	j.mode = job_mode_runnable
	jobAppend(j, g.run.prev)
}

func (g *group) priority(j *job) {
	x := j

	for x.next != g.run && j.score > x.next.score {
		x = x.next
	}

	for x.prev != g.run && j.score < x.prev.score {
		x = x.prev
	}

	jobMoveBefore(j, x)
}
