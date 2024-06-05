package scheduler

import (
	"context"
	"time"
)

type group struct {
	Group

	name string

	s *Scheduler

	block *job
	run   *job

	ctx    context.Context
	cancel context.CancelFunc

	tick chan time.Time

	nowNum  int
	runNum  int
	errNum  int
	oldRun  int
	oldErr  int
	waitNum int

	loadTime time.Duration
	loadStat statRow
}

func (g *group) init(s *Scheduler, name string) *group {
	g.s = s
	g.name = name

	g.tick = make(chan time.Time)

	g.block = &job{}
	g.block.next = g.block
	g.block.prev = g.block

	g.run = &job{}
	g.run.next = g.run
	g.run.prev = g.run

	g.loadStat.init(g.s.statSize)

	g.ctx, g.cancel = context.WithCancel(g.s.ctx)

	return g
}

func (g *group) dispatch() bool {
	if g.nowNum >= int(g.WorkerNum) {
		return false
	}

	o, err := g.GetOrder()
	if err != nil {
		if err == empty {
			if g.nowNum == 0 {
				g.waitNum = 0
				g.s.log.Println("Group", g.name, "Empty")
			}
			return false
		}
		g.s.log.Println("GetTask Error", err)

		return false
	}

	g.s.orders[o] = struct{}{}

	g.waitNum--
	g.nowNum++
	o.job.nowNum++
	g.modeCheck(o.job)

	go o.run()

	return true
}

func (g *group) end(o *order) {
	if o.err != nil {
		g.errNum += 1
		o.job.errNum += 1
	}

	loadTime := g.s.now.Sub(o.statTime)
	useTime := g.s.now.Sub(o.startTime)

	j := o.job
	j.nowNum--
	j.runNum++
	j.lastTime = g.s.now
	j.loadTime += loadTime
	j.useTime.push(int64(useTime))

	g.nowNum--
	g.runNum++
	g.loadTime += loadTime

	delete(g.s.orders, o)

	o.cancel()

	g.modeCheck(o.job)
}

func (g *group) modeCheck(j *job) {
	if j.len() == 0 {
		if j.mode != job_mode_idle {
			jobRemove(j)
			j.s.idleAdd(j)
			j.g = nil
		}
	} else if j.nowNum >= int32(j.parallel) {
		if j.mode != job_mode_block {
			jobRemove(j)
			g.blockAdd(j)
		}
	} else {
		if j.mode != job_mode_runnable {
			jobRemove(j)
			g.runAdd(j)
		}

		g.countScore(j)
		g.priority(j)
	}
}

func (g *group) GetOrder() (*order, error) {
	for {
		if g.run == g.run.next {
			return nil, empty
		}

		j := g.run.next

		taskid := j.popTask()

		if taskid == 0 {
			g.modeCheck(j)
			continue
		}

		var o = order{
			id:        taskid,
			g:         g,
			job:       j,
			startTime: g.s.now,
			statTime:  g.s.now,
			log:       g.s.log,
			dirver:    g.s.dirver,
		}

		o.ctx, o.cancel = context.WithCancel(g.ctx)

		return &o, nil
	}
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

func (g *group) countScore(j *job) {
	var x, y, z, area float64

	area = 10000

	if g.WorkerNum > 0 {
		x = float64(j.nowNum) * (area / float64(g.WorkerNum))
	}

	if g.loadStat.getAll() > 0 {
		y = float64(j.loadStat.getAll()) / float64(g.loadStat.getAll()) * area
	}

	xx := g.waitNum
	if xx > 0 {
		z = area - float64(j.len())/float64(xx)*area
	}

	j.score = int(x + y + z + float64(j.priority))
}
