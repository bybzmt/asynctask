package scheduler

import (
	"context"
	"sync"
	"time"
)

type group struct {
	GroupConfig

	s *Scheduler

	l sync.Mutex

    ctx context.Context
    cancel context.CancelFunc

	orders map[*order]struct{}
	jobs jobs

	running  bool
	complete chan *order
	tick     chan time.Time
	cmd      chan int

	now time.Time

	nowNum int
	runNum int
	errNum int
	oldRun int
	oldErr int
	waitNum int

	loadTime time.Duration
	loadStat statRow
}

func (g *group) init() {
	g.complete = make(chan *order)
	g.tick = make(chan time.Time)
	g.cmd = make(chan int)

	g.jobs.init(100, g)

	g.orders = make(map[*order]struct{})

	g.loadStat.init(g.s.statSize)

    g.ctx, g.cancel = context.WithCancel(context.Background())
}

func (g *group) dispatch() bool {
	g.l.Lock()
	defer g.l.Unlock()

    l := uint32(len(g.orders))

    if l >= g.WorkerNum {
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

	t.StartTime = g.now
	t.StatTime = g.now

    t.ctx, t.cancel = context.WithCancel(g.ctx)

	g.orders[t] = struct{}{}

	//总状态
	g.nowNum++

	go t.Run()

    return l+1 < g.WorkerNum
}

func (g *group) end(t *order) {

	g.l.Lock()
	defer g.l.Unlock()

	g.now = time.Now()

	t.EndTime = g.now

	t.logTask()

	loadTime := t.EndTime.Sub(t.StatTime)
	useTime := t.EndTime.Sub(t.StartTime)

	if t.Err != nil {
        g.errNum += 1
		t.job.errNum += 1
	}

	g.jobs.end(t.job, loadTime, useTime)

	g.runNum++
	g.nowNum--
	g.loadTime += loadTime

	delete(g.orders, t)
}

func (g *group) Run() {
	g.init()

	g.s.Log.Debugln("scheduler group", g.Id, "run")

	g.running = true

	for {
		select {
		case t := <-g.complete:
			g.end(t)

		case now := <-g.tick:
			g.statTick(now)

		case <-g.cmd:
			g.l.Lock()
			g.running = false
			g.WorkerNum = 0
			g.l.Unlock()
		}

		if g.running {
			for g.dispatch() {
            }
		} else {
			if len(g.orders) == 0 {
				break
			}
		}
	}
}

func (g *group) close() {
	g.cmd <- 1
}

func (g *group) allTaskCancel() {
    g.cancel()
}

func (g *group) statTick(now time.Time) {
	g.l.Lock()
	defer g.l.Unlock()

	g.now = now

	for t := range g.orders {
		us := now.Sub(t.StatTime)
		t.StatTime = now

		g.loadTime += us
		t.job.loadTime += us
	}

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
	for j := g.jobs.idle.next; j != g.jobs.idle; j = j.next {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}
}

func (g *group) delJob(j *job) {
	g.l.Lock()
	defer g.l.Unlock()

	g.jobs.remove(j)
}

func (g *group) addJob(j *job) {
	g.l.Lock()
	defer g.l.Unlock()

	g.jobs.addJob(j)
}

func (g *group) dayChange() {
	g.l.Lock()
	defer g.l.Unlock()

	g.oldRun = g.runNum
    g.oldErr = g.errNum
	g.runNum = 0
    g.errNum = 0
}
