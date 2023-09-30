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
	Id      ID
	Group   ID
	Name    string
	Task    string
	UseTime int
}

type TaskStat struct {
	Name     string
	RunNum   int
	ErrNum   int
	OldNum   int
	WaitNum  int
	UseTime  int
	LastTime int
	Jobs     []JobStat
}

type JobStat struct {
	JobConfig
	Group  ID
	Load   int64
	NowNum int
	Score  int
}

type GroupStat struct {
	GroupConfig
	Capacity int64
	Load     int64
	NowNum   int
	RunNum   int
	OldNum   int
}

type Statistics struct {
	schedulerConfig
	Groups    []GroupStat
	Tasks     []TaskStat
	Runs      []RunTaskStat
	Capacity  int64
	Load      int64
	NowNum    int
	RunNum    int
	ErrNum    int
	OldNum    int
	WaitNum   int
	WorkerNum uint32
	Timed     int
}

func (s *group) getStatData() (GroupStat, []RunTaskStat, map[string]JobStat) {
	s.l.Lock()
	defer s.l.Unlock()

	t := GroupStat{}
	t.GroupConfig = s.GroupConfig
	t.Capacity = int64(len(s.loadStat.data)) * int64(s.s.statTick) * int64(s.WorkerNum)
	t.Load = s.loadStat.getAll()
	t.RunNum = s.runNum
	t.OldNum = s.oldNum
	t.NowNum = s.nowNum

	now := time.Now()

	runs := make([]RunTaskStat, 0, s.WorkerNum)

	for t2 := range s.orders {
		st := RunTaskStat{
			Id:      t2.Id,
			Group:   s.Id,
			Name:    t2.Task.Name,
			Task:    t2.taskTxt(),
			UseTime: int(now.Sub(t2.StartTime) / time.Millisecond),
		}
		runs = append(runs, st)
	}

	jobs := make(map[string]JobStat, len(s.jobs.all))

	for _, j := range s.jobs.all {
		tmp := JobStat{
			JobConfig: j.JobConfig,
			Group:     s.Id,
			Load:      j.loadStat.getAll(),
			NowNum:    j.nowNum,
			Score:     j.score,
		}

		name := j.task.name

		jobs[name] = tmp
	}

	return t, runs, jobs
}
