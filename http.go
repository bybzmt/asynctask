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

var addr = flag.String("addr", ":http", "listen addr:port")
var mode = flag.String("mode", "http", "http or cmd mode")
var base = flag.String("base", os.Getenv("base"), "base url or cmd base [ENV]")

var timeout = flag.Int("timeout", 300, "task timeout Second")
var workerNum = flag.Int("num", 10, "worker number")
var parallel = flag.Int("parallel", 5, "one task parallel number")
var logfile = flag.String("log", os.Getenv("log"), "log file [ENV]")
var dbfile = flag.String("dbfile", os.Getenv("dbfile"), "storage file [ENV]")
var max_mem = flag.Uint64("max_mem", 128, "max memory size(MB)")

var redis_host = flag.String("redis_host", os.Getenv("redis_host"), "redis host [ENV]")
var redis_pwd = flag.String("redis_pwd", os.Getenv("redis_pwd"), "redis password [ENV]")
var redis_db = flag.Int("redis_db", 0, "redis database")
var redis_key = flag.String("redis_key", os.Getenv("redis_key"), "redis list key. [ENV] json: {id:uint, name:string, params:[]string, add_time:uint}")
var redis_id_key = flag.String("redis_id_key", os.Getenv("redis_id_key"), "redis task id key [ENV]")

var hub *Scheduler

//go:embed res/*
var uifiles embed.FS

func main() {
	flag.Parse()

	var logger *log.Logger

	if *dbfile == "" {
		*dbfile = "./asynctask.db"
	}

	if *logfile == "" {
		logger = log.Default()
	} else {
		fh, err := os.OpenFile(*logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		defer fh.Close()

		logger = log.New(fh, "", log.LstdFlags)
	}

	if *mode != "cmd" && *mode != "http" {
		log.Fatalln("parameter mode must be cmd or http")
	}
	if *workerNum < 1 {
		log.Fatalln("parameter num must > 0")
	}
	if *timeout < 1 {
		log.Fatalln("parameter timeout must > 0")
	}
	if *parallel < 1 {
		log.Fatalln("parameter parallel must >= 1")
	}

	env := new(Environment).Init(*workerNum, *base, *timeout, logger)
	env.DbFile = *dbfile
	env.Parallel = *parallel

	if *mode == "cmd" {
		env.Mode = MODE_CMD
	} else {
		env.Mode = MODE_HTTP
	}

	hub = new(Scheduler).Init(env)

	tfs, _ := fs.Sub(uifiles, "res")
	http.Handle("/", http.FileServer(http.FS(tfs)))

	http.HandleFunc("/api/status", page_status)
	http.HandleFunc("/api/task/add", page_task_add)

	go func() {
		log.Fatalln(http.ListenAndServe(*addr, nil))
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
		Name:    name,
		Params:  data,
		AddTime: time.Now().Unix(),
	}

	ok := hub.AddOrder(o)

	rs := &Result{Code: 0, Data: "ok"}
	if !ok {
		rs.Code = 1
		rs.Data = "Add Fail"
	}

	json.NewEncoder(w).Encode(rs)
}

func page_status(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	t := hub.Status()

	rs := &Result{Code: 0, Data: t}
	json.NewEncoder(w).Encode(rs)
}
