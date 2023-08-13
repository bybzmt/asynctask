package tool

import (
	"asynctask/scheduler"
	"embed"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"io/fs"
)

var hub *scheduler.Scheduler
var addr string

//go:embed dist/*
var uifiles embed.FS

type Result struct {
	Code int
	Data interface{}
}

func init_http() {
	tfs, _ := fs.Sub(uifiles, "dist")
	http.Handle("/", http.FileServer(http.FS(tfs)))

	http.HandleFunc("/api/status", page_status)
	http.HandleFunc("/api/task/add", page_task_add)
	http.HandleFunc("/api/task/cancel", page_task_cancel)
	http.HandleFunc("/api/job/empty", page_job_empty)
	http.HandleFunc("/api/job/priority", page_job_priority)
	http.HandleFunc("/api/job/parallel", page_job_parallel)
	http.HandleFunc("/api/job/delIdle", page_job_delIdle)
}

func page_task_add(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	data, err := ioutil.ReadAll(r.Body)
    if err != nil {
        //todo log
        w.WriteHeader(400)
        return
    }

	o := scheduler.Task{}
    err = json.Unmarshal(data, &o)
    if err != nil {
        //todo log
        w.WriteHeader(400)
        return
    }

	err = addOrder(&o)

	rs := &Result{Code: 0, Data: "ok"}
	if err != nil {
		rs.Code = 1
		rs.Data = err.Error()
	}

	json.NewEncoder(w).Encode(rs)
}

func page_status(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	t := hub.GetStatData()

	rs := &Result{Code: 0, Data: t}
	json.NewEncoder(w).Encode(rs)
}

func page_job_empty(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

    sid, _ := strconv.Atoi(r.FormValue("sid"))
    jid, _ := strconv.Atoi(r.FormValue("jid"))

	err := hub.JobEmpty(scheduler.ID(sid), scheduler.ID(jid))

	rs := &Result{Code: 0, Data: "ok"}

    if err != nil {
        rs.Code = 1
		rs.Data = err.Error()
    }

	json.NewEncoder(w).Encode(rs)
}

func page_job_delIdle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

    sid, _ := strconv.Atoi(r.FormValue("sid"))
    jid, _ := strconv.Atoi(r.FormValue("jid"))

	err := hub.JobDelIdle(scheduler.ID(sid), scheduler.ID(jid))

	rs := &Result{Code: 0, Data: "ok"}

    if err != nil {
        rs.Code = 1
		rs.Data = err.Error()
    }

	json.NewEncoder(w).Encode(rs)
}

func page_job_priority(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

    sid, _ := strconv.Atoi(r.FormValue("sid"))
    jid, _ := strconv.Atoi(r.FormValue("jid"))
	priority, _ := strconv.Atoi(r.FormValue("priority"))

	err := hub.JobPriority(scheduler.ID(sid), scheduler.ID(jid), priority)

	rs := &Result{Code: 0, Data: "ok"}

    if err != nil {
        rs.Code = 1
		rs.Data = err.Error()
    }

	json.NewEncoder(w).Encode(rs)
}

func page_job_parallel(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

    sid, _ := strconv.Atoi(r.FormValue("sid"))
    jid, _ := strconv.Atoi(r.FormValue("jid"))
	parallel, _ := strconv.Atoi(r.FormValue("parallel"))

	err := hub.JobParallel(scheduler.ID(sid), scheduler.ID(jid), uint32(parallel))

	rs := &Result{Code: 0, Data: "ok"}

    if err != nil {
        rs.Code = 1
		rs.Data = err.Error()
    }

	json.NewEncoder(w).Encode(rs)
}

func page_task_cancel(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

    sid, _ := strconv.Atoi(r.FormValue("sid"))
	tid, _ := strconv.Atoi(r.FormValue("tid"))

	err := hub.OrderCancel(scheduler.ID(sid), scheduler.ID(tid))

	rs := &Result{Code: 0, Data: "ok"}

    if err != nil {
        rs.Code = 1
		rs.Data = err.Error()
    }

	json.NewEncoder(w).Encode(rs)
}


func InitHub(h *scheduler.Scheduler) {
    hub = h
}

func HttpRun(addr string) {
    init_http()

	hub.Log.Fatalln(http.ListenAndServe(addr, nil))
}
