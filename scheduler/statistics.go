package scheduler

import (
	"time"
)

type statRow struct {
	offset int
	all    int64
	data   []int64
}

func (s *statRow) init(num int) {
	s.offset = 0
	s.all = 0
	s.data = make([]int64, 0, num)
}

func (s *statRow) push(val int64) {
	if len(s.data) < cap(s.data) {
		s.data = append(s.data, 0)
	}

	s.all += val - s.data[s.offset]
	s.data[s.offset] = val
	s.offset = (s.offset + 1) % cap(s.data)
}

func (s *statRow) getAll() int64 {
	return s.all
}

func (s *statRow) len() int64 {
	return int64(len(s.data))
}

type RunTask struct {
	Id        ID
	Group     string
	Job       string
	StartTime int64
}

type JobStat struct {
	Priority int32
	Parallel uint32
	Name     string
	RunNum   int
	ErrNum   int
	OldRun   int
	OldErr   int
	WaitNum  int
	UseTime  int
	LastTime int64
	Group    string
	Load     int64
	NowNum   int
	Score    int
}

type GroupStat struct {
	Name      string
	WorkerNum uint32
	Note      string

	Capacity int64
	Load     int64
	NowNum   int
	RunNum   int
	ErrNum   int
	OldRun   int
	OldErr   int
	WaitNum  int
}

type Stat struct {
	Groups []GroupStat
	Tasks  []JobStat
}

func getJobStat(j *job) JobStat {
	useTime := 0
	if j.useTime.len() > 0 {
		useTime = int(j.useTime.getAll() / j.useTime.len() / int64(time.Millisecond))
	}

	var lastTime int64
	if !j.lastTime.IsZero() {
		lastTime = j.lastTime.Unix()
	}

	tmp := JobStat{
		Priority: j.priority,
		Parallel: j.parallel,
		Name:     j.name,
		Group:    j.group,
		WaitNum:  int(j.len()),
		NowNum:   int(j.nowNum),
		RunNum:   int(j.runNum),
		ErrNum:   int(j.errNum),
		OldRun:   int(j.oldRun),
		OldErr:   int(j.oldErr),
		UseTime:  useTime,
		LastTime: lastTime,
		Load:     j.loadStat.getAll() / int64(time.Millisecond),
		Score:    j.score,
	}

	return tmp
}

func getGroupStat(g *group) (t GroupStat) {
	t.WorkerNum = g.WorkerNum
	t.Note = g.Note
	t.Name = g.name

	t.Capacity = g.loadStat.len() * int64(time.Second) * int64(g.WorkerNum) / int64(time.Millisecond)
	t.Load = g.loadStat.getAll() / int64(time.Millisecond)
	t.NowNum = g.nowNum
	t.RunNum = g.runNum
	t.ErrNum = g.errNum
	t.OldRun = g.oldRun
	t.OldErr = g.oldErr
	t.WaitNum = g.waitNum
	return
}

func (s *Scheduler) GetRunTask() []RunTask {
	s.l.Lock()
	defer s.l.Unlock()

	runs := make([]RunTask, 0, len(s.orders))

	for t2 := range s.orders {
		st := RunTask{
			Id:        t2.id,
			Group:     t2.g.name,
			Job:       t2.job.name,
			StartTime: t2.startTime.Unix(),
		}
		runs = append(runs, st)
	}

	return runs
}

func (s *Scheduler) GetStat() Stat {
	s.l.Lock()
	defer s.l.Unlock()

	var out Stat

	out.Groups = make([]GroupStat, 0, len(s.groups))
	out.Tasks = make([]JobStat, 0, len(s.jobs))

	for _, jt := range s.jobs {
		out.Tasks = append(out.Tasks, getJobStat(jt))
	}

	for _, g := range s.groups {
		out.Groups = append(out.Groups, getGroupStat(g))
	}

	return out
}
