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

var ts_rand chan int

var to = flag.String("to", "", "to asynctask")
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
		ac := "ac" + strconv.Itoa(an)
		sl := ts_actions[ac]
		sl = ts_getRand() % sl

		p := url.Values{}
		p.Add("code", "200")
		p.Add("sleep", strconv.Itoa(sl))

		v := url.Values{}
		v.Add("action", ac)
		v.Add("params", p.Encode())

		resp, err := http.PostForm(*to, v)
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
