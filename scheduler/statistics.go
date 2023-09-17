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

type StatTask struct {
	Id      ID
	Name    string
	Task    string
	UseTime int
}

type JobStat struct {
	JobConfig
	Name     string
	Load     int64
	NowNum   int
	RunNum   int
	ErrNum   int
	OldNum   int
	WaitNum  int
	UseTime  int
	LastTime int
	Score    int
}

type Statistics struct {
    GroupConfig
	Tasks    []StatTask
	Jobs     []JobStat
	Capacity int64
	Load     int64
	NowNum   int
	RunNum   int
	ErrNum   int
	OldNum   int
	WaitNum  int
}

func (s *group) getStatData() *Statistics {
	s.l.Lock()
	defer s.l.Unlock()

	t := &Statistics{}
    t.GroupConfig = s.GroupConfig
    t.Capacity = int64(len(s.loadStat.data)) * int64(s.s.statTick) * int64(s.WorkerNum)
	t.Jobs = make([]JobStat, 0, s.jobs.len())
	t.Tasks = make([]StatTask, 0, len(s.orders))
	t.Load = s.loadStat.getAll()
	t.RunNum = s.runNum
	t.OldNum = s.oldNum
	t.NowNum = s.nowNum
	t.WaitNum = s.waitNum

	now := time.Now()

	for t2 := range s.orders {
		st := StatTask{
			Id:      t2.Id,
			Name:    t2.Task.Name,
            Task:    t2.taskTxt(),
			UseTime: int(now.Sub(t2.StartTime) / time.Millisecond),
		}
		t.Tasks = append(t.Tasks, st)
	}

	for _, j := range s.jobs.all {

		useTime := 0
		if len(j.useTimeStat.data) > 0 {
			useTime = int(j.useTimeStat.getAll() / int64(len(j.useTimeStat.data)) / int64(time.Millisecond))
		}

		sec := 0

		sec2 := j.lastTime.Unix()
		if sec2 > 0 {
			sec = int(now.Sub(j.lastTime) / time.Second)
		}

		t.Jobs = append(t.Jobs, JobStat{
			JobConfig: j.JobConfig,
			Name:      j.task.name,
			Load:      j.loadStat.getAll(),
			RunNum:    j.runNum(),
			OldNum:    j.oldNum(),
			NowNum:    j.nowNum,
			ErrNum:    j.errNum(),
			WaitNum:   j.waitNum(),
			UseTime:   useTime,
			LastTime:  sec,
			Score:     j.score,
		})
	}

	return t
}
