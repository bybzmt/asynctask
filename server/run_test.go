package server

import (
	"asynctask/scheduler"
	"context"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
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

func TestRun(t *testing.T) {
	var hub Server
	var my myServer

	taskadd := make(chan int, 10)
	taskend := make(chan int, 10)

	go initTestServer(&my, taskend)
	defer my.Close()

	time.Sleep(time.Millisecond * 100)

	to := "http://" + my.l.Addr().String()
	to = strings.ReplaceAll(to, "[::]", "127.0.0.1")

	initServer(&hub, to)

	hub.HttpEnable = true
	hub.Http.Addr = "127.0.0.1:8080"

	if err := hub.Init(); err != nil {
		panic(err)
	}

	ctx, canceler := context.WithCancel(context.Background())

	go hub.Run(ctx)

	logrus.Println("listen", to)
	logrus.Println("http", hub.Http.Addr)

	go addTask(&hub, 10000, taskadd)

	httpNum := 0
	allNum := 0
	runnum := 0
	oldTrigger := 0

	for {
		select {
		case x := <-taskadd:
			allNum++
			if x == 2 {
				httpNum++
			}

		case trigger := <-taskend:

			if trigger > 0 {
				if trigger >= oldTrigger {
					oldTrigger = trigger
				} else {
					t.Error("timer task order error")
				}
			}

			runnum++

			if runnum == httpNum {
				goto toend
			}
		}
	}

toend:

	logrus.Println("all task end")

	time.Sleep(time.Millisecond * 200)

	stat := hub.Scheduler.GetStatData()

	canceler()

	if stat.Timed != 0 {
		t.Error("timer task not empty num:", stat.Timed)
	}

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
		logrus.Println("run task", RunNum, "/", allNum)
	}
}

func addTask(hub *Server, num int, taskadd chan int) {

	for i := 0; i < num; i++ {
		if i%1000 == 0 {
			logrus.Println("i:", i, "/", num)
		}

		an := ts_getRand() % len(ts_actions)
		sl := ts_actions[an]

		var task scheduler.Task

		if ts_getRand()%(num/100) == 0 {
			task.Timer = uint(time.Now().Unix()) + 2
		}

		if ts_getRand()%7 == 0 {
			task.Name = "cli://test/echo"
			task.Args, _ = json.Marshal([]string{"1"})

			taskadd <- 1
		} else {

			tmp := ts_getRand()
			sleep := tmp % sl

			p := url.Values{}
			p.Add("code", "200")
			p.Add("sleep", strconv.Itoa(sleep))

			if task.Timer > 0 {
				p.Add("trigger", strconv.Itoa(int(task.Timer)))
			}

			if task.Timer > 0 {
				task.Name = "http://trigger/" + strconv.Itoa(sl)
			} else if sl > 1000 {
				task.Name = "http://slow/" + strconv.Itoa(sl)
			} else {
				task.Name = "http://fast/" + strconv.Itoa(sleep)
			}

			task.Name += "/?" + p.Encode()

			taskadd <- 2
		}

		err := hub.Scheduler.TaskAdd(&task)
		if err != nil {
			panic(err)
		}
	}

	logrus.Println("i:", num, "/", num)
}

func ts_getRand() int {
	b := make([]byte, 4)
	rand.Read(b)
	num := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
	return num
}

type myServer struct {
	http.Server
	l net.Listener
}

func initTestServer(my *myServer, taskend chan int) {
	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		panic(err)
	}

	my.l = l

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s_code := r.FormValue("code")
		s_sleep := r.FormValue("sleep")
		s_trigger := r.FormValue("trigger")

		code, _ := strconv.Atoi(s_code)
		sleep, _ := strconv.Atoi(s_sleep)
		trigger, _ := strconv.Atoi(s_trigger)

		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}

		w.WriteHeader(code)
		w.Write([]byte(http.StatusText(code)))

		taskend <- trigger
	})

	my.Serve(my.l)
}

func initServer(hub *Server, to string) {

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	hub.Scheduler.WorkerNum = 10
	hub.Scheduler.Parallel = 1

	f, err := os.CreateTemp("", "asynctask_*.bolt")
	if err != nil {
		panic(err)
	}

	logrus.Println("tmpfile", f.Name())

	defer os.Remove(f.Name())
	defer f.Close()

	db, err := bolt.Open(f.Name(), 0644, nil)
	if err != nil {
		panic(err)
	}

	hub.Scheduler.Db = db

	err = hub.Scheduler.Init()
	if err != nil {
		panic(err)
	}

	err = hub.Scheduler.GroupAdd(scheduler.GroupConfig{
		Note:      "test_group2",
		WorkerNum: 10,
	})
	if err != nil {
		panic(err)
	}

	//------ 2 groups -----

	r := scheduler.TaskRule{
		Pattern:     "^cli://test/",
		RewriteReg:  "^cli://test/(.+)",
		RewriteRepl: "$1",
		Type:        1,
		Sort:        1,
		Used:        true,
	}

	err = hub.Scheduler.TaskRulePut(r)
	if err != nil {
		panic(err)
	}

	r = scheduler.TaskRule{
		Pattern:     "^http://slow",
		RewriteReg:  "^http://slow/(.+)",
		RewriteRepl: to + "/$1",
		Type:        1,
		Sort:        2,
		Used:        true,
	}

	err = hub.Scheduler.TaskRulePut(r)
	if err != nil {
		panic(err)
	}

    jr := scheduler.JobRule{
		Pattern:     "^http://slow",
		Type:        1,
		Sort:        2,
		Used:        true,
	}
    jr.Parallel = 10
    jr.GroupId = 1

	err = hub.Scheduler.JobRulePut(jr)
	if err != nil {
		panic(err)
	}

	err = hub.Scheduler.TaskRulePut(r)
	if err != nil {
		panic(err)
	}

	r = scheduler.TaskRule{
		Pattern:     "^http://fast",
		RewriteReg:  "^http://fast/(.+)",
		RewriteRepl: to + "/$1",
		Type:        1,
		Sort:        3,
		Used:        true,
	}

	err = hub.Scheduler.TaskRulePut(r)
	if err != nil {
		panic(err)
	}

	r = scheduler.TaskRule{
		Pattern:     "^(http://trigger)",
		RewriteReg:  "^http://trigger/(.+)",
		RewriteRepl: to + "/$1",
		Type:        1,
		Sort:        4,
		Used:        true,
	}

	err = hub.Scheduler.TaskRulePut(r)
	if err != nil {
		panic(err)
	}
}
