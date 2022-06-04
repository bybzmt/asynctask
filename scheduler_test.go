package main

import (
	"bufio"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

var ts_actions = map[string]int{
	"ac0":  5,
	"ac1":  5,
	"ac2":  10,
	"ac3":  10,
	"ac4":  50,
	"ac5":  100,
	"ac6":  200,
	"ac7":  500,
	"ac8":  1000,
	"ac9":  2000,
	"ac10": 4000,
}

var ts_action_num = 1000
var ts_action_now int64 = 0
var ts_close chan bool
var ts_rand = make(chan int, 100)

func TestScheduler(t *testing.T) {
	l, addr, err := ts_Listen()
	if err != nil {
		t.Fatal("listen error")
		return
	}

	log.Println("listen on:", addr)

	ts_close = make(chan bool)

	go ts_server(l)

	baseurl := "http://127.0.0.1:" + addr + "/test"

	log.Println("baseurl:", baseurl)

	env := new(Config)
	env.Base = baseurl
	env.LogFile = "my_[date].log"
	env.Init("http", 10)

	hub := new(Scheduler).Init(env)

	go ts_initRand()
	go ts_addTask(hub, baseurl)

	hub.Run()
	l.Close()
}

func ts_Listen() (l net.Listener, addr string, err error) {
	for port := 8080; port < 8900; port++ {
		l, err = net.Listen("tcp", ":"+strconv.Itoa(port))
		if err == nil {
			addr = strconv.Itoa(port)
			break
		}
	}

	return l, addr, err
}

func ts_server(l net.Listener) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s_code := r.FormValue("code")
		s_sleep := r.FormValue("sleep")

		code, _ := strconv.Atoi(s_code)
		sleep, _ := strconv.Atoi(s_sleep)

		//log.Println(r.URL.Path, code, sleep)

		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}

		w.WriteHeader(code)
		w.Write([]byte(http.StatusText(code)))

		now := atomic.AddInt64(&ts_action_now, 1)
		if now >= int64(ts_action_num) {
			ts_close <- true
		}
	})

	s := &http.Server{}
	s.Serve(l)
}

func ts_addTask(hub *Scheduler, baseurl string) {
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < ts_action_num; i++ {
		an := ts_getRand() % len(ts_actions)
		ac := "ac" + strconv.Itoa(an)
		sl := ts_actions[ac]
		sl = ts_getRand() % sl

		if ts_getRand()%3 == 0 {
			ac = baseurl + ac
		} else {
			ac = "/" + ac
		}

		data := "code=200&sleep=" + strconv.Itoa(sl)

		o := Order{
			Name:    ac,
			Params:  []string{data},
			AddTime: uint(time.Now().Unix()),
		}

		hub.AddOrder(&o)
	}

	go func() {
		c := time.Tick(time.Second * 1)
		for _ = range c {
			hub.Status()
		}
	}()

	<-ts_close

	//log.Println(hub.Status())

	hub.Close()
}

func ts_getRand() int {
	return <-ts_rand
}

func ts_initRand() {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		for {
			rand.Seed(time.Now().UnixNano())
			ts_rand <- rand.Int()
		}
		return
	}
	defer f.Close()

	bf := bufio.NewReaderSize(f, 4096)

	b := make([]byte, 4)

	for {
		bf.Read(b)
		num := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
		ts_rand <- num
	}
}
