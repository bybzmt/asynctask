package main

import (
	"asynctask/scheduler"
	"asynctask/tool"
	"crypto/rand"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

var ts_actions = []int{
	5, 10, 50, 80,
	100, 110, 120, 130, 140, 150, 160, 170, 180, 190,
	100, 110, 120, 130, 140, 150, 160, 170, 180, 190,
	200, 230, 240, 270,
	300, 350, 400, 500,
	1000,
	3000,
	6000,
}

func TestRun(t *testing.T) {
	var hub *scheduler.Scheduler
	var my myServer

	taskadd := make(chan int, 10)
	taskend := make(chan int, 10)

	go initTestServer(&my, taskend)

	initHub(&hub)
	go hub.Run()
	go tool.HttpRun(hub, ":8080")

	time.Sleep(time.Millisecond * 100)

	to := "http://" + my.l.Addr().String()
	log.Println("listen", to)
	log.Println("http", ":8080")

	go addTask(hub, 5000, to, taskadd)

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

            // log.Println("taskend", runnum, httpNum)

			if runnum == httpNum {
				goto toend
			}
		}
	}

toend:

	log.Println("taskend all")

	time.Sleep(time.Millisecond * 200)

	stat := hub.GetStatData()

	hub.Close()

	if stat.Timed != 0 {
		t.Error("timer task not empty num:", stat.Timed)
	}

	if stat.WaitNum != 0 {
		t.Error("task not empty num:", stat.WaitNum)
	}

	if stat.RunNum != allNum {
		t.Error("run task num err", stat.RunNum, "/", allNum)
	}

	my.Close()
}

func addTask(hub *scheduler.Scheduler, num int, to string, taskadd chan int) {

	for i := 0; i < num; i++ {
		if i%1000 == 0 {
			log.Println("i:", i, "/", num)
		}

		an := ts_getRand() % len(ts_actions)
		sl := ts_actions[an]

		var task scheduler.Task

		if ts_getRand()%11 == 0 {
			task.Trigger = uint(time.Now().Unix()) + 2
		}

		if ts_getRand()%7 == 0 {
			task.Name = "cli/" + strconv.Itoa(sl)
			task.Cli = &scheduler.TaskCli{
				Cmd:    "echo",
				Params: []string{task.Name},
			}
			taskadd <- 1
		} else {

			tmp := ts_getRand()
			sleep := tmp % sl

			p := url.Values{}
			p.Add("code", "200")
			p.Add("sleep", strconv.Itoa(sleep))
			p.Add("trigger", strconv.Itoa(int(task.Trigger)))

			l := to + "/?" + p.Encode()

			if task.Trigger > 0 {
				task.Name = "http/trigger"
			} else if sl > 1000 {
				task.Name = "slow/" + strconv.Itoa(sl)
			} else {
				task.Name = "http/" + strconv.Itoa(sleep)
			}

			task.Http = &scheduler.TaskHttp{
				Url: l,
			}

			taskadd <- 2
		}

		err := hub.AddTask(&task)
		if err != nil {
			panic(err)
		}
	}

	log.Println("i:", num, "/", num)
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

func initHub(hub **scheduler.Scheduler) {

	logrus.SetLevel(logrus.DebugLevel)
	cfg.Log = logrus.StandardLogger()

	cfg.WorkerNum = 10
	cfg.Parallel = 1

	f, err := os.CreateTemp("", "asynctask_*.bolt")
	if err != nil {
		panic(err)
	}

	log.Println("tmpfile", f.Name())

	defer os.Remove(f.Name())
	defer f.Close()

	db, err := bolt.Open(f.Name(), 0644, nil)
	if err != nil {
		panic(err)
	}

	cfg.Db = db

	*hub, err = scheduler.New(cfg)
	if err != nil {
		panic(err)
	}

	gc, err := (*hub).AddGroup()
	if err != nil {
		panic(err)
	}

	gc.Note = "test_group2"

	err = (*hub).SetGroupConfig(gc)
	if err != nil {
		panic(err)
	}

	//------ 2 groups -----

	rc, err := (*hub).AddRoute()
	if err != nil {
		panic(err)
	}

	rc.Note = "http_slow_router"
	rc.Match = `^slow/.+`
	rc.Parallel = 5
	rc.Sort = 2
	rc.GroupId = gc.Id
	rc.Used = true

	err = (*hub).SetRouteConfig(rc)
	if err != nil {
		panic(err)
	}

	//------ cli -----

	rc, err = (*hub).AddRoute()
	if err != nil {
		panic(err)
	}

	rc.Note = "cli_router"
	rc.Match = `^cli/.+`
	rc.Parallel = 1
	rc.Sort = 3
	rc.GroupId = gc.Id
	rc.Mode = scheduler.MODE_CLI
	rc.Used = true

	err = (*hub).SetRouteConfig(rc)
	if err != nil {
		panic(err)
	}

}
