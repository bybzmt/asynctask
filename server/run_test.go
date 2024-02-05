package server

import (
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
	var hub *Server
	var my myServer

	taskadd := make(chan int, 10)
	taskend := make(chan int, 10)

	go initTestServer(&my, taskend)
	defer my.Close()

	time.Sleep(time.Millisecond * 100)

	to := "http://" + my.l.Addr().String()
	to = strings.ReplaceAll(to, "[::]", "127.0.0.1")

	hub, err := initServer(to)
	if err != nil {
		panic(err)
	}

	go hub.Start()

	logrus.Println("listen", to)
	logrus.Println("http", hub.cfg.HttpAddr)

	go addTask(hub, 10000, taskadd)

	allNum := 0
	timerNum := 0
	runnum := 0
	oldTrigger := 0

	tick := time.NewTimer(time.Second)

	t.Log("runnum", runnum, allNum)

	for {
		select {
		case <-tick.C:
			t.Log("runnum", runnum, allNum)

		case x := <-taskadd:
			allNum++

			if x == 2 {
				timerNum++
			}

		case trigger := <-taskend:
			t.Log("taskend", runnum, timerNum, allNum)

			if trigger > 0 {
				if trigger >= oldTrigger {
					oldTrigger = trigger
				} else {
					t.Error("timer task order error")
				}
			}

			runnum++

			if runnum == allNum {
				goto toend
			}
		}
	}

toend:

	logrus.Println("all task end")

	time.Sleep(time.Millisecond * 200)

	stat := hub.s.GetStat()
	hub.Stop()

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

		var task Task

		if (ts_getRand() % 10000) < 100 {
			task.RunAt = time.Now().Unix() + 2
		}

		tmp := ts_getRand()
		sleep := tmp % sl

		p := url.Values{}
		p.Add("code", "200")
		p.Add("sleep", strconv.Itoa(sleep))

		if task.RunAt > 0 {
			p.Add("trigger", strconv.Itoa(int(task.RunAt)))

			taskadd <- 2
		} else {
			taskadd <- 1
		}

		if task.RunAt > 0 {
			task.Url = "trigger/" + strconv.Itoa(sl)
		} else if sl > 1000 {
			task.Url = "slow/" + strconv.Itoa(sl)
		} else {
			task.Url = "fast/" + strconv.Itoa(sleep)
		}

		task.Url += "/?" + p.Encode()

		err := hub.TaskAdd(&task)
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

func initServer(to string) (*Server, error) {

	cfg := Config{
		HttpEnable: true,
		HttpAddr:   "127.0.0.1:8080",
		Timeout:    600,
		Jobs: []*Job{
			{
				Pattern:  "^slow",
				Parallel: 10,
				Group:    "slow",
			},
		},
		Groups: map[string]*Group{
			"slow": {
				WorkerNum: 10,
				Note:      "slow",
			},
		},

		Routes: []*Route{
			{
				Pattern: "^fast/(.+)",
				Job:     "fast/$1",
				Dirver:  "http",
				Rewrite: &Rewrite{
					Pattern: "^fast/",
					Rewrite: to + "/",
				},
			},
			{
				Pattern: "^slow/",
				Job:     "slow",
				Dirver:  "http",
				Rewrite: &Rewrite{
					Pattern: "^slow/",
					Rewrite: to + "/",
				},
			},
			{
				Pattern: "trigger/(.+)",
				Job:     "trigger/$1",
				Dirver:  "http",
				Rewrite: &Rewrite{
					Pattern: "^trigger/",
					Rewrite: to + "/",
				},
			},
		},
		Dirver: map[string]*Dirver{
			"http": {
				Type: DIRVER_HTTP,
			},
		},
	}

	f, err := os.CreateTemp("", "asynctask_*.bolt")
	if err != nil {
		panic(err)
	}

	defer os.Remove(f.Name())
	defer f.Close()

	f2, err := os.CreateTemp("", "config_*.json")
	if err != nil {
		panic(err)
	}

	defer os.Remove(f2.Name())
	defer f2.Close()

	err = json.NewEncoder(f2).Encode(&cfg)
	if err != nil {
		panic(err)
	}

	err = f2.Sync()
	if err != nil {
		panic(err)
	}

	l := logrus.StandardLogger()
	l.SetLevel(logrus.DebugLevel)
	l.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	return New(f2.Name(), f.Name(), l)
}
