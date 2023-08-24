package tool

import (
	"asynctask/scheduler"
	"embed"
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"net/http"
	"strconv"
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

	http.HandleFunc("/api/status", page_error(page_status))
	http.HandleFunc("/api/task/add", page_error(page_task_add))
	http.HandleFunc("/api/task/cancel", page_error(page_task_cancel))
	http.HandleFunc("/api/job/emptyAll", page_error(page_job_empty))
	http.HandleFunc("/api/job/delIdle", page_error(page_job_delIdle))
	http.HandleFunc("/api/job/setConfig", page_error(page_job_config))
	http.HandleFunc("/api/group/setConfig", page_error(page_group_config))
	http.HandleFunc("/api/router/setConfig", page_error(page_router_config))
}

func page_status(r *http.Request) any {
	return hub.GetStatData()
}

func page_task_add(r *http.Request) any {
	var o scheduler.Task

	if err := httpReadJson(r, &o); err != nil {
		return err
	}

	return addOrder(&o)
}

func page_job_empty(r *http.Request) any {
	sid, _ := strconv.Atoi(r.FormValue("sid"))
	jid, _ := strconv.Atoi(r.FormValue("jid"))

	return hub.JobEmpty(scheduler.ID(sid), scheduler.ID(jid))
}

func page_job_delIdle(r *http.Request) any {
	sid, _ := strconv.Atoi(r.FormValue("sid"))
	jid, _ := strconv.Atoi(r.FormValue("jid"))

	return hub.JobDelIdle(scheduler.ID(sid), scheduler.ID(jid))
}

func page_job_config(r *http.Request) any {

	name := r.FormValue("name")
	var cfg scheduler.JobConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return hub.SetJobConfig(name, cfg)
}

func page_group_config(r *http.Request) any {

	gid, _ := strconv.Atoi(r.FormValue("gid"))
	var cfg scheduler.GroupConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return hub.SetGroupConfig(scheduler.ID(gid), cfg)
}

func page_router_config(r *http.Request) any {

	rid, _ := strconv.Atoi(r.FormValue("rid"))
	var cfg scheduler.RouterConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return hub.SetRouterConfig(scheduler.ID(rid), cfg)
}

func page_task_cancel(r *http.Request) any {
	gid, _ := strconv.Atoi(r.FormValue("gid"))
	oid, _ := strconv.Atoi(r.FormValue("oid"))

	return hub.OrderCancel(scheduler.ID(gid), scheduler.ID(oid))
}

func InitHub(h *scheduler.Scheduler) {
	hub = h
}

func HttpRun(addr string) {
	init_http()

	hub.Log.Fatalln(http.ListenAndServe(addr, nil))
}

func httpReadJson(r *http.Request, out any) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, out)
	if err != nil {
		return err
	}

	return nil
}

func page_error(fn func(r *http.Request) any) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		var rs Result

		defer func() {
			if err := recover(); err != nil {
				rs.Code = 1
				rs.Data = err
			}

			w.Header().Add("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rs)
		}()

		rs.Data = fn(r)

		if _, ok := rs.Data.(error); ok {
			rs.Code = 1
		}
	}
}
