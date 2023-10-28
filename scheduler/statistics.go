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
	JobConfig
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

func (g *group) getJobStat(jt *job) JobStat {
	useTime := 0
	if len(jt.useTimeStat.data) > 0 {
		useTime = int(jt.useTimeStat.getAll() / int64(len(jt.useTimeStat.data)) / int64(time.Millisecond))
	}

	sec := 0

	sec2 := jt.lastTime.Unix()
	if sec2 > 0 {
		sec = int(g.s.now.Sub(jt.lastTime) / time.Second)
	}

	tmp := JobStat{
		JobConfig: jt.JobConfig,
		Name:      jt.Name,
		WaitNum:   int(jt.waitNum),
		NowNum:    int(jt.nowNum),
		RunNum:    int(jt.runNum),
		ErrNum:    int(jt.errNum),
		OldRun:    int(jt.oldRun),
		OldErr:    int(jt.oldErr),
		UseTime:   useTime,
		LastTime:  sec,
		GroupId:   jt.group.Id,
		Load:      jt.loadStat.getAll(),
		Score:     jt.score,
	}

	return tmp
}

func (s *group) getGroupStat() GroupStat {
	t := GroupStat{}
	t.GroupConfig = s.GroupConfig
	t.Capacity = int64(len(s.loadStat.data)) * int64(s.s.statTick) * int64(s.WorkerNum)
	t.Load = s.loadStat.getAll()
	t.NowNum = s.nowNum
	t.RunNum = s.runNum
	t.ErrNum = s.errNum
	t.OldRun = s.oldRun
	t.OldErr = s.oldErr
	t.WaitNum = s.waitNum

	return t
}

func (s *Scheduler) getRunTaskStat() []RunTaskStat {

	runs := make([]RunTaskStat, 0, s.WorkerNum)

	for t2 := range s.orders {
		st := RunTaskStat{
			Id:        t2.Id,
			Group:     t2.g.Id,
			Name:      t2.job.Name,
			Task:      t2.taskTxt,
			StartTime: t2.startTime.Unix(),
		}
		runs = append(runs, st)
	}

	return runs
}
