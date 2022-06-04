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
	"strconv"
	"syscall"
	"time"
)

var addr = flag.String("addr", ":http", "listen addr:port")
var mode = flag.String("mode", "http", "http or cmd mode")
var base = flag.String("base", os.Getenv("base"), "base url or cmd base [ENV]")

var timeout = flag.Int("timeout", 300, "task timeout Second")
var workerNum = flag.Int("num", 10, "worker number")
var parallel = flag.Int("parallel", 5, "one task default parallel")
var logfile = flag.String("log", os.Getenv("log"), "log file [ENV]")
var dbfile = flag.String("dbfile", os.Getenv("dbfile"), "storage file [ENV]")
var max_mem = flag.Uint("max_mem", 128, "max memory size(MB)")

var redis_host = flag.String("redis_host", os.Getenv("redis_host"), "redis host [ENV]")
var redis_pwd = flag.String("redis_pwd", os.Getenv("redis_pwd"), "redis password [ENV]")
var redis_db = flag.String("redis_db", os.Getenv("redis_db"), "redis database [ENV]")
var redis_key = flag.String("redis_key", os.Getenv("redis_key"), "redis list key. [ENV] \njson: {id:uint, parallel:uint, name:string, params:[]string, add_time:uint}")

var hub *Scheduler

//go:embed res/*
var uifiles embed.FS

func main() {
	flag.Parse()

	if *dbfile == "" {
		*dbfile = "./asynctask.db"
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

	env := new(Config)
	env.Parallel = uint(*parallel)
	env.WorkerNum = *workerNum
	env.Base = *base
	env.DbFile = *dbfile
	env.LogFile = *logfile
	env.MaxMem = *max_mem

	env.RedisDb = *redis_db
	env.RedisKey = *redis_key
	env.RedisHost = *redis_host
	env.RedisPwd = *redis_pwd

	env.Init(*mode, *timeout)

	hub = new(Scheduler).Init(env)

	tfs, _ := fs.Sub(uifiles, "res")
	http.Handle("/", http.FileServer(http.FS(tfs)))

	http.HandleFunc("/api/status", page_status)
	http.HandleFunc("/api/task/add", page_task_add)

	go func() {
		log.Fatalln(http.ListenAndServe(*addr, nil))
	}()

	go exitSignal()

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

	_id := r.FormValue("id")
	name := r.FormValue("action")
	data := r.Form["params"]

	id, _ := strconv.Atoi(_id)
	parallel, _ := strconv.Atoi(r.FormValue("parallel"))

	o := &Order{
		Id:       uint(id),
		Parallel: uint(parallel),
		Name:     name,
		Params:   data,
		AddTime:  uint(time.Now().Unix()),
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
