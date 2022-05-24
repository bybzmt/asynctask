package main

import (
	"time"
)

type StatRow struct {
	offset int
	all    int64
	data   []int64
}

func (s *StatRow) Init(num int) {
	s.data = make([]int64, num)
}

func (s *StatRow) Push(val int64) {
	s.all += val - s.data[s.offset]
	s.data[s.offset] = val
	s.offset = (s.offset + 1) % len(s.data)
}

func (s *StatRow) GetNow() int64 {
	return s.data[s.offset]
}

func (s *StatRow) GetAll() int64 {
	return s.all
}

type Stat struct {
	Name    string
	Load    int
	NowNum  int
	RunNum  int
	OldNum  int
	WaitNum int
	UseTime int
}

type Statistics struct {
	All  Stat
	Jobs []Stat
}

func (s *Scheduler) Status() *Statistics {
	s.cmd <- 3
	t := <-s.statResp
	return t
}

func (s *Scheduler) getStatData() {
	e1 := s.LoadStat.GetAll() + s.IdleStat.GetAll()
	all := 0
	if e1 > 0 {
		all = int(float64(s.LoadStat.GetAll()) / float64(s.LoadStat.GetAll()+s.IdleStat.GetAll()) * 10000)
	}

	t := &Statistics{}
	t.Jobs = make([]Stat, 0, s.jobs.Len())
	t.All.Name = "all"
	t.All.Load = all
	t.All.RunNum = s.RunNum
	t.All.OldNum = s.OldNum
	t.All.NowNum = s.NowNum
	t.All.WaitNum = s.WaitNum

	s.jobs.Each(func(j *Job) {
		x := 0
		if e1 > 0 {
			x = int(float64(j.LoadStat.GetAll()) / float64(e1) * 10000)
		}

		t.Jobs = append(t.Jobs, Stat{
			Name:    j.Name,
			Load:    x,
			RunNum:  j.RunNum,
			OldNum:  j.OldNum,
			NowNum:  j.NowNum,
			WaitNum: j.Len(),
			UseTime: int(j.UseTimeStat.GetAll()/int64(time.Millisecond)) / len(j.UseTimeStat.data),
		})
	})

	s.statResp <- t
}
