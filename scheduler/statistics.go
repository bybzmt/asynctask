package scheduler

import (
	"time"
)

type StatRow struct {
	offset int
	all    int64
	data   []int64
}

func (s *StatRow) Init(num int) {
	s.offset = 0
	s.all = 0
	s.data = make([]int64, 0, num)
}

func (s *StatRow) Push(val int64) {
	if len(s.data) < cap(s.data) {
		s.data = append(s.data, 0)
	}

	s.all += val - s.data[s.offset]
	s.data[s.offset] = val
	s.offset = (s.offset + 1) % cap(s.data)
}

func (s *StatRow) GetAll() int64 {
	return s.all
}

type StatTask struct {
	Id      ID
	Name    string
	UseTime int
}

type JobStat struct {
    JobConfig
	Name     string
	Load     int
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
    Id ID
    Config GroupConfig
	All   JobStat
	Tasks []StatTask
	Jobs  []JobStat
}

func (s *group) getStatData() *Statistics {
    s.l.Lock()
    defer s.l.Unlock()

	e1 := float64(len(s.LoadStat.data) * int(s.s.StatTick) * int(s.WorkerNum))

	all := 0
	if e1 > 0 {
		all = int(float64(s.LoadStat.GetAll()) / e1 * 10000)
	}

	t := &Statistics{}
    t.Id = s.id
    t.Config = s.GroupConfig
	t.Jobs = make([]JobStat, 0, s.jobs.Len())
	t.Tasks = make([]StatTask, 0, len(s.orders))
	t.All.Name = "all"
	t.All.Load = all
	t.All.RunNum = s.RunNum
	t.All.OldNum = s.OldNum
	t.All.NowNum = s.NowNum
	t.All.WaitNum = s.WaitNum

	now := time.Now()

	for t2 := range s.orders {
		st := StatTask{
			Id:      t2.Id,
			Name:    t2.job.Name,
			UseTime: int(now.Sub(t2.StartTime) / time.Millisecond),
		}
		t.Tasks = append(t.Tasks, st)
	}

    for _, j := range s.jobs.all {

        x := 0
        if e1 > 0 {
            x = int(float64(j.LoadStat.GetAll()) / e1 * 10000)
        }

        useTime := 0
        if len(j.UseTimeStat.data) > 0 {
            useTime = int(j.UseTimeStat.GetAll() / int64(len(j.UseTimeStat.data)) / int64(time.Millisecond))
        }

        sec := 0

        sec2 := j.LastTime.Unix()
        if sec2 > 0 {
            sec = int(now.Sub(j.LastTime) / time.Second)
        }

        t.Jobs = append(t.Jobs, JobStat{
            JobConfig: j.JobConfig,
            Name:     j.Name,
            Load:     x,
            RunNum:   j.runNum(),
            OldNum:   j.oldNum(),
            NowNum:   j.NowNum,
            ErrNum:   j.errNum(),
            WaitNum:  j.waitNum(),
            UseTime:  useTime,
            LastTime: sec,
            Score:    j.Score,
        })
    }

	return t
}

