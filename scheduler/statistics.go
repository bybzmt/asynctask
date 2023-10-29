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

type RunTaskStat struct {
	Id        ID
	Group     ID
	Name      string
	Mode      string
	Task      string
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
	LastTime int
	GroupId  ID
	Load     int64
	NowNum   int
	Score    int
}

type GroupStat struct {
	GroupConfig
	Capacity int64
	Load     int64
	NowNum   int
	RunNum   int
	ErrNum   int
	OldRun   int
	OldErr   int
	WaitNum  int
}

type Statistics struct {
	schedulerConfig
	Groups []GroupStat
	Tasks  []JobStat
	Timed  int
}

func (g *group) getJobStat(j *job) JobStat {
	useTime := 0
	if len(j.useTimeStat.data) > 0 {
		useTime = int(j.useTimeStat.getAll() / int64(len(j.useTimeStat.data)) / int64(time.Millisecond))
	}

	sec := 0

	sec2 := j.lastTime.Unix()
	if sec2 > 0 {
		sec = int(g.s.now.Sub(j.lastTime) / time.Second)
	}

	tmp := JobStat{
		Priority: j.Priority,
		Parallel: j.Parallel,
		Name:     j.name,
		WaitNum:  int(j.waitNum),
		NowNum:   int(j.nowNum),
		RunNum:   int(j.runNum),
		ErrNum:   int(j.errNum),
		OldRun:   int(j.oldRun),
		OldErr:   int(j.oldErr),
		UseTime:  useTime,
		LastTime: sec,
		GroupId:  j.group.Id,
		Load:     j.loadStat.getAll(),
		Score:    j.score,
	}

	return tmp
}

func (s *Scheduler) getGroupStat(g *group) (t GroupStat) {
	t.GroupConfig = g.GroupConfig
	t.Capacity = int64(len(g.loadStat.data)) * int64(s.statTick) * int64(g.WorkerNum)
	t.Load = g.loadStat.getAll()
	t.NowNum = g.nowNum
	t.RunNum = g.runNum
	t.ErrNum = g.errNum
	t.OldRun = g.oldRun
	t.OldErr = g.oldErr
	t.WaitNum = g.waitNum
	return
}

func (s *Scheduler) GetRunTaskStat() []RunTaskStat {
	s.l.Lock()
	defer s.l.Unlock()

	runs := make([]RunTaskStat, 0, s.WorkerNum)

	for t2 := range s.orders {
		st := RunTaskStat{
			Id:        t2.Id,
			Group:     t2.g.Id,
			Name:      t2.job.name,
			Task:      t2.taskTxt,
			StartTime: t2.startTime.Unix(),
		}
		runs = append(runs, st)
	}

	return runs
}

func (s *Scheduler) GetStatData() Statistics {
	s.l.Lock()
	defer s.l.Unlock()

	var out Statistics
	out.schedulerConfig = s.schedulerConfig
	out.Timed = s.timedNum

	out.Groups = make([]GroupStat, 0, len(s.groups))
	out.Tasks = make([]JobStat, 0, len(s.jobs))

	for _, jt := range s.jobs {
		tmp := jt.group.getJobStat(jt)

		out.Tasks = append(out.Tasks, tmp)
	}

	for _, g := range s.groups {
		out.Groups = append(out.Groups, s.getGroupStat(g))
	}

	return out
}

func (s *Scheduler) GetGroupStat() (out []GroupStat) {
	s.l.Lock()
	defer s.l.Unlock()

	for _, g := range s.groups {
		out = append(out, s.getGroupStat(g))
	}

	return
}
