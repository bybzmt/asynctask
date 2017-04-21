package main

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
