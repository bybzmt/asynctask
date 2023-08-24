package scheduler

import (
	"container/list"
	"encoding/json"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

type group struct {
	GroupConfig

	id ID

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

	today int
	now   time.Time

	workerId int

	//己执行任务计数
	RunNum int
	//昨天任务计数
	OldNum int
	//执行中的任务
	NowNum int
	//总队列长度
	WaitNum int

	//负载数据
	LoadTime time.Duration
	LoadStat StatRow
}

func (g *group) init(s *Scheduler) error {
    g.s = s

	g.complete = make(chan *order)

	g.jobs.init(100, g)

	g.workers.Init()
	g.orders = make(map[*order]struct{})

	g.LoadStat.Init(g.s.StatSize)

	if err := g.loadJobs(); err != nil {
		return err
	}

	return nil
}

func (g *group) loadJobs() error {

	//key: task/:gid/:jname
	err := g.s.Db.View(func(tx *bolt.Tx) error {
		bucket, err := getBucketMust(tx, "task", fmtId(g.id))
		if err != nil {
			return err
		}

		return bucket.ForEachBucket(func(k []byte) error {
			g.jobs.addJob(string(k))

			return nil
		})
	})

	return err
}

func (g *group) addOrder(o *order) error {
	g.l.Lock()
	defer g.l.Unlock()

	err := g.jobs.addOrder(o)
	if err != nil {
		return err
	}

	g.WaitNum++

	return nil
}

func (g *group) workerNumCheck() {
	for len(g.allWorkers) != int(g.WorkerNum) {
		if len(g.allWorkers) < int(g.WorkerNum) {
			g.workerId++

			w := new(worker)
			w.Init(g.workerId, g)

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

	g.workerNumCheck()

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
	g.NowNum++
	g.WaitNum--

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
		t.job.ErrNum++
	}

	g.jobs.end(t.job, loadTime, useTime)

	g.RunNum++
	g.NowNum--
	g.LoadTime += loadTime

	g.workers.PushBack(t.worker)
	t.worker = nil

	delete(g.orders, t)
}

func (g *group) Run() {
	g.s.Log.Debugln("scheduler group", g.id, "run")

	g.today = time.Now().Day()
	g.running = true

	for {
		select {
		case t := <-g.complete:
			g.end(t)

		case now := <-g.tick:
			g.statTick(now)
		}

		if g.running {
			g.dispatch()
		} else {
			if len(g.allWorkers) == 0 {
				break
			}
		}
	}

	g.s.end <- g
}

func (g *group) close() {
	g.WorkerNum = 0

	for _, w := range g.allWorkers {
		w.Close()
	}
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
		us := g.now.Sub(t.StatTime)
		t.StatTime = g.now

		g.LoadTime += us
		t.job.LoadTime += us
	}

	g.LoadStat.Push(int64(g.LoadTime))
	g.LoadTime = 0

	for _, j := range g.jobs.all {
		j.LoadStat.Push(int64(j.LoadTime))
		j.LoadTime = 0
	}

	g.dayCheck()
}

func (g *group) dayCheck() {
	if g.today != g.now.Day() {
		g.OldNum = g.RunNum
		g.RunNum = 0

		for _, j := range g.jobs.all {
			j.OldNum = j.RunNum
			j.RunNum = 0
			j.ErrNum = 0
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
		Name:     t.job.Name,
		Status:   t.Status,
		WaitTime: logSecond(waitTime),
		RunTime:  logSecond(runTime),
		Output:   t.Msg,
	}

	msg, _ := json.Marshal(d)

	g.s.Log.Infoln("[Task] %s\n", msg)
}