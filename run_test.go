package main

import (
	"asynctask/scheduler"
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
	var rund chan int
	var num = 1000
	var my myServer
	sleep := 0

	rund = make(chan int)

	go initTestServer(&my, rund)
	go initHub(&hub)

	time.Sleep(time.Millisecond * 100)

	to := "http://" + my.l.Addr().String()
	log.Println("listen", to)

	for i := 0; i < num; i++ {
		if i%1000 == 0 {
			log.Println("i:", i, "/", num)
		}

		an := ts_getRand() % len(ts_actions)
		sl := ts_actions[an]
		ac := "/ac" + strconv.Itoa(sl)
		tmp := ts_getRand()
		sl = tmp % sl

		p := url.Values{}
		p.Add("code", "200")
		p.Add("sleep", strconv.Itoa(sl))

		l := to + "/?" + p.Encode()

		var task scheduler.Task
		task.Name = ac
		task.Http = &scheduler.TaskHttp{
			Url: l,
		}

		if sl > 1000 {
			task.Trigger = uint(time.Now().Unix()) + 2
			sleep++
		}

		err := hub.AddTask(&task)
		if err != nil {
			panic(err)
		}
	}

	log.Println("wait run", sleep)

	runnum := 0

	for {
		<-rund

		runnum++

		if runnum == num {
			break
		}
	}

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
	f.Close()

	db, err := bolt.Open(f.Name(), 0644, nil)
	if err != nil {
		panic(err)
	}

	cfg.Db = db

	*hub, err = scheduler.New(cfg)
	if err != nil {
		panic(err)
	}

	(*hub).Run()
}
