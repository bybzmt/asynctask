package main

import (
	"asynctask/scheduler"
	"log"
    "crypto/rand"
	"net/http"
	"net/url"
    "net"
	"os"
	"strconv"
	"time"
    "testing"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
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

var ts_rand chan int

var num = 1000
var sleep = 10
var my myServer
var hub *scheduler.Scheduler
var runnum int
var rund chan int

func TestRun(t *testing.T) {
    go initTestServer()
    go initHub()

    time.Sleep(time.Millisecond*100)

    to := "http://" + my.l.Addr().String()
    log.Println("listen", to)

	ts_rand = make(chan int, 10)
	rund = make(chan int, 10)

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

		l := to + "?" + p.Encode()

        var task scheduler.Task
        task.Name = ac
        task.Http = &scheduler.TaskHttp{
            Url : l,
        }

        err := hub.AddTask(&task)
        if err != nil {
            panic(err)
        }
	}

    log.Println("wait run")

    for {
        <-rund

        runnum++

        if runnum == num {
            break
        }
    }

    my.Close()

    hub.Close()
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

func initTestServer() {
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

func initHub() {

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

	hub, err = scheduler.New(cfg)
	if err != nil {
        panic(err)
	}

	hub.Run()
}
