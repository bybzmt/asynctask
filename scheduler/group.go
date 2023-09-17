package scheduler

import (
	"container/list"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

type group struct {
	GroupConfig

	s *Scheduler

	l sync.Mutex

	//所有工作进程
	allWorkers []*worker
	//空闲工作进程
	workers list.List
	//运行中的任务
	orders map[*order]struct{}
	//所有任务
	jobs jobs

	running  bool
	complete chan *order
	tick     chan time.Time
	cmd      chan int

	today int
	now   time.Time

	//己执行任务计数
	runNum int
	//昨天任务计数
	oldNum int
	//执行中的任务
	nowNum int
	//总队列长度
	waitNum int

	//负载数据
	loadTime time.Duration
	loadStat statRow
}

func (g *group) init() {
	g.complete = make(chan *order)
	g.tick = make(chan time.Time)
	g.cmd = make(chan int)

	g.jobs.init(100, g)

	g.workers.Init()
	g.orders = make(map[*order]struct{})

	g.loadStat.init(g.s.statSize)
}

func (g *group) workerNumCheck() {
	g.l.Lock()
	defer g.l.Unlock()

	for len(g.allWorkers) != int(g.WorkerNum) {
		if len(g.allWorkers) < int(g.WorkerNum) {

			id := atomic.AddUint32((*uint32)(&g.s.WorkerId), 1)

			w := new(worker)
			w.Init(ID(id), g)

			g.allWorkers = append(g.allWorkers, w)
			g.workers.PushBack(w)

			go w.Run()
		} else {
			ew := g.workers.Back()
			if ew == nil {
				return
			}

			g.workers.Remove(ew)
			w := ew.Value.(*worker)

			w.Close()

			workers := make([]*worker, 0, g.WorkerNum)
			for _, t := range g.allWorkers {
				if t != w {
					workers = append(workers, t)
				}
			}
			g.allWorkers = workers
		}
	}
}

func (g *group) dispatch() {
	g.l.Lock()
	defer g.l.Unlock()

	//得到工人
	ew := g.workers.Back()
	if ew == nil {
		return
	}

	t, err := g.jobs.GetOrder()
	if err != nil {
		if err == Empty {
			return
		}
		g.s.Log.Warnln("GetTask Error", err)

		return
	}

	t.StartTime = g.now
	t.StatTime = g.now

	g.orders[t] = struct{}{}

	g.workers.Remove(ew)
	w := ew.Value.(*worker)

	//总状态
	g.nowNum++
	g.waitNum--

	//分配任务
	t.worker = w

	w.Exec(t)
}

func (g *group) end(t *order) {
	g.l.Lock()
	defer g.l.Unlock()

	g.now = time.Now()

	t.EndTime = g.now

	g.logTask(t)

	loadTime := t.EndTime.Sub(t.StatTime)
	useTime := t.EndTime.Sub(t.StartTime)

	if t.Err != nil {
		t.job.errAdd()
	}

	g.jobs.end(t.job, loadTime, useTime)

	g.runNum++
	g.nowNum--
	g.loadTime += loadTime

	g.workers.PushBack(t.worker)
	t.worker = nil

	delete(g.orders, t)
}

func (g *group) Run() {
    g.init()

	g.s.Log.Debugln("scheduler group", g.Id, "run")

	g.today = time.Now().Day()
	g.running = true

	for {
		select {
		case t := <-g.complete:
			g.end(t)

		case now := <-g.tick:
			g.statTick(now)
			g.workerNumCheck()

		case <-g.cmd:
			g.l.Lock()
			g.running = false
			g.WorkerNum = 0
			g.l.Unlock()

			g.workerNumCheck()
		}

		if g.running {
			g.dispatch()
		} else {
			if len(g.allWorkers) == 0 {
				break
			}
		}
	}

	g.s.groupEnd <- g
}

func (g *group) close() {
	g.cmd <- 1
}

func (g *group) allTaskCancel() {
	for t := range g.orders {
		t.worker.Cancel()
	}
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

	for _, j := range g.jobs.all {
		j.loadStat.push(int64(j.loadTime))
		j.loadTime = 0
	}

	g.dayCheck()
}

func (g *group) dayCheck() {
	if g.today != g.now.Day() {
		g.oldNum = g.runNum
		g.runNum = 0

		for _, j := range g.jobs.all {
			j.dayChange()
		}

		g.today = g.now.Day()
	}
}

func (g *group) logTask(t *order) {

	var waitTime float64 = 0
	if t.AddTime.Unix() > 0 {
		waitTime = t.StartTime.Sub(t.AddTime).Seconds()
	}

	runTime := t.EndTime.Sub(t.StartTime).Seconds()

	d := taskLog{
		Id:       t.Id,
		Name:     t.job.task.name,
		Task:     t.taskTxt(),
		Status:   t.Status,
		WaitTime: logSecond(waitTime),
		RunTime:  logSecond(runTime),
		Output:   t.Msg,
	}

	msg, _ := json.Marshal(d)

	g.s.Log.Infoln("[Task] %s\n", msg)
}

func (g *group) notifyDelJob(jname string) {
	g.l.Lock()
	defer g.l.Unlock()

	j, ok := g.jobs.all[jname]
	if ok {
		g.jobs.remove(j)
		delete(g.jobs.all, jname)
	}
}

func (g *group) notifyJob(jtask *jobTask) {
	g.l.Lock()
	defer g.l.Unlock()

	j := g.jobs.addJob(jtask)

	g.jobs.modeCheck(j)
}
