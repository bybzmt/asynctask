package scheduler

import (
	"context"
	"time"
)

type group struct {
	GroupConfig

	s *Scheduler

	ctx    context.Context
	cancel context.CancelFunc

	jobs jobs

	running bool
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

	g.jobs.init()

	g.loadStat.init(g.s.statSize)

	g.ctx, g.cancel = context.WithCancel(context.Background())
}

func (g *group) dispatch() bool {
	if g.nowNum >= int(g.WorkerNum) {
		return false
	}

	t, err := g.jobs.GetOrder()
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

	t.StartTime = g.s.now
	t.StatTime = g.s.now

	t.ctx, t.cancel = context.WithCancel(g.ctx)

	g.s.orders[t] = struct{}{}

	//总状态
	g.nowNum++

	go t.Run()

	return g.nowNum+1 < int(g.WorkerNum)
}

func (g *group) end(t *order) {
	loadTime := t.EndTime.Sub(t.StatTime)
	useTime := t.EndTime.Sub(t.StartTime)

	if t.Err != nil {
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

			g.s.timerAddTask(t.Task)
		}
	}

	g.jobs.end(t.job, loadTime, useTime)

	g.runNum++
	g.nowNum--
	g.loadTime += loadTime

	delete(g.s.orders, t)
}

func (g *group) statMaintain() {
	g.loadStat.push(int64(g.loadTime))
	g.loadTime = 0

	for j := g.jobs.run.next; j != g.jobs.run; j = j.next {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}

	for j := g.jobs.block.next; j != g.jobs.block; j = j.next {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}
}

func (s *Scheduler) statMaintain(now time.Time) {
    for t := range s.orders {
        us := now.Sub(t.StatTime)
        t.StatTime = now

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
