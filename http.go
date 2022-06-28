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
	"strings"
	"syscall"
	"time"
)

var addr = flag.String("addr", ":http", "listen addr:port")
var mode = flag.String("mode", "http", "http or cmd mode")

var timeout = flag.Int("timeout", 300, "task timeout Second")

var cfg Config
var hub Scheduler

func init() {
	flag.StringVar(&cfg.Base, "base", os.Getenv("base"), "base url or cmd base [ENV]")
	flag.IntVar(&cfg.WorkerNum, "num", 10, "worker number")
	flag.UintVar(&cfg.Parallel, "parallel", 5, "one task default parallel")
	flag.StringVar(&cfg.LogFile, "log", os.Getenv("log"), "log file e.g: my-[date].log [ENV]")
	flag.StringVar(&cfg.DbFile, "dbfile", os.Getenv("dbfile"), "storage file [ENV]")
	flag.UintVar(&cfg.MaxTask, "max_task", 1000000, "max task num")

	flag.StringVar(&cfg.RedisHost, "redis_host", os.Getenv("redis_host"), "redis host [ENV]")
	flag.StringVar(&cfg.RedisPwd, "redis_pwd", os.Getenv("redis_pwd"), "redis password [ENV]")
	flag.StringVar(&cfg.RedisDb, "redis_db", os.Getenv("redis_db"), "redis database [ENV]")
	flag.StringVar(&cfg.RedisKey, "redis_key", os.Getenv("redis_key"), "redis list key. [ENV] \njson: {id:uint, parallel:uint, name:string, params:[]string, add_time:uint}")
}

//go:embed dist/*
var uifiles embed.FS

func main() {
	flag.Parse()

	if *mode != "cmd" && *mode != "http" {
		log.Fatalln("parameter mode must be cmd or http")
	}
	if *timeout < 1 {
		log.Fatalln("parameter timeout must > 0")
	}
	if cfg.WorkerNum < 1 {
		log.Fatalln("parameter num must > 0")
	}
	if cfg.Parallel < 1 {
		log.Fatalln("parameter parallel must >= 1")
	}

	cfg.Init(*mode, *timeout)

	hub.Init(&cfg)

	tfs, _ := fs.Sub(uifiles, "dist")
	http.Handle("/", http.FileServer(http.FS(tfs)))

	http.HandleFunc("/api/status", page_status)
	http.HandleFunc("/api/task/add", page_task_add)
	http.HandleFunc("/api/task/cancel", page_task_cancel)
	http.HandleFunc("/api/job/empty", page_job_empty)
	http.HandleFunc("/api/job/priority", page_job_priority)
	http.HandleFunc("/api/job/parallel", page_job_parallel)
	http.HandleFunc("/api/job/delIdle", page_job_delIdle)

	go func() {
		log.Fatalln(http.ListenAndServe(*addr, nil))
	}()

	go exitSignal()

	hub.Run()
}

func exitSignal() {
	co := make(chan os.Signal, 1)
	signal.Notify(co, os.Interrupt, syscall.SIGTERM)
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
	name := r.FormValue("name")
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

func page_job_empty(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	name := r.FormValue("name")
	name = strings.TrimSpace(name)

	ok := hub.JobEmpty(name)

	rs := &Result{Code: 0, Data: ok}
	json.NewEncoder(w).Encode(rs)
}

func page_job_delIdle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	name := r.FormValue("name")
	name = strings.TrimSpace(name)

	ok := hub.JobDelIdle(name)

	rs := &Result{Code: 0, Data: ok}
	json.NewEncoder(w).Encode(rs)
}

func page_job_priority(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	name := strings.TrimSpace(r.FormValue("name"))
	tmp := strings.TrimSpace(r.FormValue("priority"))

	priority, _ := strconv.Atoi(tmp)

	ok := hub.JobPriority(name, priority)

	rs := &Result{Code: 0, Data: ok}
	json.NewEncoder(w).Encode(rs)
}

func page_job_parallel(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	name := strings.TrimSpace(r.FormValue("name"))
	tmp := strings.TrimSpace(r.FormValue("parallel"))

	parallel, _ := strconv.Atoi(tmp)
	if tmp == "-" {
		parallel = int(hub.cfg.Parallel)
	}

	ok := hub.JobParallel(name, parallel)

	rs := &Result{Code: 0, Data: ok}
	json.NewEncoder(w).Encode(rs)
}

func page_task_cancel(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	id := r.FormValue("id")

	ok := hub.taskCancel(id)

	rs := &Result{Code: 0, Data: ok}
	json.NewEncoder(w).Encode(rs)
}
