package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var workerNum = flag.Int("num", 10, "worker number")
var baseurl = flag.String("baseurl", "", "base url")
var addr = flag.String("addr", ":http", "listen addr:port")

var hub *Scheduler

//go:embed res/*
var uifiles embed.FS

func main() {
	flag.Parse()

	std := log.New(os.Stdout, "[Info] ", log.LstdFlags)
	err := log.New(os.Stderr, "[Scheduler] ", log.LstdFlags)

	env := new(Environment).Init(*workerNum, *baseurl, std, err)

	hub = new(Scheduler).Init(env)

	tfs, _ := fs.Sub(uifiles, "res")
	http.Handle("/", http.FileServer(http.FS(tfs)))

	http.HandleFunc("/api/status", page_status)
	http.HandleFunc("/api/task/add", page_task_add)

	go func() {
		log.Fatal(http.ListenAndServe(*addr, nil))
	}()

	go exitSignal()
	go hub.restoreFromFile()
	go redis_init()

	hub.Run()
}

func exitSignal() {
	co := make(chan os.Signal, 1)
	signal.Notify(co, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-co

	hub.Close()
}

type Result struct {
	Code int
	Data interface{}
}

func page_task_add(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	name := r.FormValue("action")
	data := r.Form["params"]

	o := &Order{
		Id:      0,
		Task:    name,
		Params:  data,
		AddTime: time.Now().Unix(),
	}

	hub.AddOrder(o)

	rs := &Result{Code: 0, Data: "ok"}
	json.NewEncoder(w).Encode(rs)
}

func page_status(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	t := hub.Status()

	rs := &Result{Code: 0, Data: t}
	json.NewEncoder(w).Encode(rs)
}
