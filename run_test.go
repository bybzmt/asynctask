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

	num := 0

	for range taskadd {
		num++
	}

	log.Println("taskadd", num)

	runnum := 0

	for {
		<-taskend

		runnum++
		log.Println("taskend", runnum)

		if runnum == num {
			break
		}
	}

	log.Println("taskend all")

	stat := hub.GetStatData()

	if stat.Timed != 0 {
		t.Error("timer task not empty num:", stat.Timed)
	}

	for stat.WaitNum != 0 {
		t.Error("task not empty num:", stat.WaitNum)
	}

	hub.Close()
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

		if ts_getRand()%5 == 0 {
			task.Trigger = uint(time.Now().Unix()) + 2
		}

		if ts_getRand()%7 == 0 {
			task.Name = "cli/" + strconv.Itoa(sl)
			task.Cli = &scheduler.TaskCli{
				Cmd:    "echo",
				Params: []string{task.Name},
			}
		} else {

			tmp := ts_getRand()
			sleep := tmp % sl

			p := url.Values{}
			p.Add("code", "200")
			p.Add("sleep", strconv.Itoa(sleep))

			l := to + "/?" + p.Encode()

			if sl > 1000 {
				task.Name = "slow/" + strconv.Itoa(sl)
			} else {
				task.Name = "http/" + strconv.Itoa(sl)
			}

			task.Http = &scheduler.TaskHttp{
				Url: l,
			}

			taskadd <- 1
		}

		err := hub.AddTask(&task)
		if err != nil {
			panic(err)
		}
	}

	log.Println("i:", num, "/", num)

	close(taskadd)
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

func initTestServer(my *myServer, rund chan int) {
	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		panic(err)
	}

	my.l = l

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s_code := r.FormValue("code")
		s_sleep := r.FormValue("sleep")

		code, _ := strconv.Atoi(s_code)
		sleep, _ := strconv.Atoi(s_sleep)

		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}

		w.WriteHeader(code)
		w.Write([]byte(http.StatusText(code)))

		rund <- 1
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
