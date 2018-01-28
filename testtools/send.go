package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var ts_actions = []int{
	5, 10, 50, 80,
	100, 110, 120, 130, 140, 150, 160, 170, 180, 190,
	100, 110, 120, 130, 140, 150, 160, 170, 180, 190,
	200, 230, 240, 270,
	300, 350, 400, 500,
	1000,
	//3000,
	//6000,
	//10000,
}

var ts_rand chan int

var to = flag.String("to", "127.0.0.1:8080", "asynctask host")
var num = flag.Int("num", 10000, "task num")
var sleep = flag.Int("sleep", 10, "sleep ms")

func main() {
	flag.Parse()

	log.Println("runing")

	ts_rand = make(chan int, 100)

	go ts_initRand()

	ts_rand = make(chan int, 100)
	for i := 0; i < *num; i++ {
		if i%1000 == 0 {
			log.Println("i:", i, "/", *num)
		}

		an := ts_getRand() % len(ts_actions)
		sl := ts_actions[an]
		ac := "/ac" + strconv.Itoa(sl)
		tmp := ts_getRand()
		sl = tmp % sl



		p := url.Values{}
		p.Add("code", "200")
		p.Add("sleep", strconv.Itoa(sl))

		v := url.Values{}

		if tmp % 13 == 0 {
			//ac = "http://127.0.0.1:8081" + ac
		}

		if tmp % 10 == 0 {
			v.Add("method", "GET")
		} else {
			v.Add("method", "POST")
		}

		v.Add("action", ac)
		v.Add("params", p.Encode())

		resp, err := http.PostForm("http://"+*to+"/task/add", v)
		if err != nil {
			panic(err)
		}
		_, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		time.Sleep(time.Duration(*sleep) * time.Millisecond)
	}
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
