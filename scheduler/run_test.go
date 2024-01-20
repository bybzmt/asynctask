package scheduler

import (
	"context"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"
)

var ts_actions = []int{
	5, 10, 50, 80,
	100, 110, 120, 130, 140, 150, 160, 170, 180, 190,
	100, 110, 120, 130, 140, 150, 160, 170, 180, 190,
	200, 230, 240, 270,
	300, 350, 400, 500,
	1000, 1000, 1000, 2000,
	3000,
	6000,
}

type tasktask struct {
	id    ID
	code  int
	sleep int
	job   string
}

type tasks struct {
	id  ID
	l   sync.Mutex
	m   map[ID]tasktask
	add chan int
	end chan int
}

func TestRun(t *testing.T) {

	ts := tasks{}
	ts.m = make(map[ID]tasktask)
	ts.add = make(chan int, 10)
	ts.end = make(chan int, 10)

	c := Config{
		Group:       "default",
		WorkerNum:   5,
		Parallel:    1,
		JobsMaxIdle: 100,
		CloseWait:   10,

		Jobs: []*Job{
			&Job{
				Pattern:  "slow",
				Group:    "slow",
				Parallel: 10,
			},
			&Job{
				Pattern:  "^fast",
				Group:    "fast",
				Parallel: 2,
			},
		},

		Groups: map[string]*Group{
			"slow": &Group{
				Note:      "slow",
				WorkerNum: 10,
			},
		},

		Dirver: DirverFunc(func(id ID, ctx context.Context) error {

			ts.l.Lock()
			t, ok := ts.m[id]
			ts.l.Unlock()

			if !ok {
				return NotFound
			}

			if t.sleep > 0 {
				time.Sleep(time.Duration(t.sleep) * time.Millisecond)
			}

			ts.end <- 1

			return nil
		}),
	}

	s, err := New(&c)
	if err != nil {
		t.Fatal("New", err)
	}

	go s.Run()

	go addTask(s, &ts, 10000)

	allNum := 0
	runnum := 0

	for {
		select {
		case <-ts.add:
			allNum++

		case <-ts.end:
			runnum++

			if runnum == allNum {
				goto toend
			}
		}
	}

toend:

	t.Log("all task end")

	time.Sleep(time.Millisecond * 200)

	stat := s.GetStat()

	RunNum := 0
	for _, g := range stat.Groups {
		if g.WaitNum != 0 {
			t.Error("task not empty num:", g.WaitNum)
		}

		RunNum += g.RunNum
	}

	if RunNum != allNum {
		t.Error("run task num err", RunNum, "/", allNum)
	} else {
		t.Log("run task", RunNum, "/", allNum)
	}
}

func addTask(s *Scheduler, ts *tasks, num int) {

	for i := 0; i < num; i++ {
		if i%1000 == 0 {
			log.Println("i:", i, "/", num)
		}

		an := ts_getRand() % len(ts_actions)
		sl := ts_actions[an]

		ts.l.Lock()
		ts.id++

		var task tasktask

		tmp := ts_getRand()
		sleep := tmp % sl

		task.id = ts.id
		task.sleep = sleep

		if sl > 1000 {
			task.job = "slow"
			task.sleep = sleep
		} else {
			task.job = "fast/" + strconv.Itoa(sl)
			task.sleep = sleep
		}

		ts.m[ts.id] = task
		ts.add <- 1
		ts.l.Unlock()

		s.TaskAdd(&Task{Id: task.id, Job: task.job})
	}

	log.Println("i:", num, "/", num)
}
