package tool

import (
	"asynctask/scheduler"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
)

//go:embed dist/*
var uifiles embed.FS

type Result struct {
	Code int
	Data interface{}
}

type HttpServer struct {
    http.Server
    Hub *scheduler.Scheduler
}

func HttpRun(hub *scheduler.Scheduler, addr string) {
    s := HttpServer{
        Hub: hub,
    }
    s.Addr = addr

    s.Init()
	s.Hub.Log.Fatalln(s.ListenAndServe())
}

func (s *HttpServer) Init() {
    h := http.NewServeMux()

	tfs, _ := fs.Sub(uifiles, "dist")
	h.Handle("/", http.FileServer(http.FS(tfs)))

	h.HandleFunc("/api/status", page_error(s.page_status))
	h.HandleFunc("/api/task/add", page_error(s.page_task_add))
	h.HandleFunc("/api/task/cancel", page_error(s.page_task_cancel))
	h.HandleFunc("/api/job/delIdle", page_error(s.page_job_empty))
	h.HandleFunc("/api/job/emptyAll", page_error(s.page_job_empty))
	h.HandleFunc("/api/job/setConfig", page_error(s.page_job_config))
	h.HandleFunc("/api/routes", page_error(s.page_routes))
	h.HandleFunc("/api/route/add", page_error(s.page_route_add))
	h.HandleFunc("/api/route/del", page_error(s.page_route_del))
	h.HandleFunc("/api/route/setConfig", page_error(s.page_route_config))
	h.HandleFunc("/api/groups", page_error(s.page_groups))
	h.HandleFunc("/api/group/add", page_error(s.page_group_add))
	h.HandleFunc("/api/group/del", page_error(s.page_group_del))
	h.HandleFunc("/api/group/setConfig", page_error(s.page_group_config))

    s.Handler = h
}

func (s *HttpServer) page_status(r *http.Request) any {
	return s.Hub.GetStatData()
}

func (s *HttpServer) page_task_add(r *http.Request) any {
	var o scheduler.Task

	if err := httpReadJson(r, &o); err != nil {
		return err
	}

	return s.Hub.AddTask(&o)
}

func (s *HttpServer) page_job_empty(r *http.Request) any {
	name := r.FormValue("name")

	return s.Hub.JobEmpty(name)
}

func (s *HttpServer) page_job_delIdle(r *http.Request) any {
	gid, _ := strconv.Atoi(r.FormValue("gid"))
	jname := strings.TrimSpace(r.FormValue("name"))

	return s.Hub.JobDelIdle(scheduler.ID(gid), jname)
}

func (s *HttpServer) page_job_config(r *http.Request) any {
	jname := strings.TrimSpace(r.FormValue("name"))
    _cfg := r.FormValue("cfg")

	var cfg scheduler.JobConfig

    if err := json.Unmarshal([]byte(_cfg), &cfg); err != nil {
        return err
    }

	return s.Hub.SetJobConfig(jname, cfg)
}

func (s *HttpServer) page_groups(r *http.Request) any {
    return s.Hub.GetGroupConfigs()
}

func (s *HttpServer) page_group_add(r *http.Request) any {
    cfg, err := s.Hub.AddGroup()
    if err != nil {
        return err
    }
    return cfg
}

func (s *HttpServer) page_group_del(r *http.Request) any {

	gid, _ := strconv.Atoi(r.FormValue("gid"))

	return s.Hub.DelGroup(scheduler.ID(gid))
}

func (s *HttpServer) page_group_config(r *http.Request) any {

	var cfg scheduler.GroupConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Hub.SetGroupConfig(cfg)
}

func (s *HttpServer) page_routes(r *http.Request) any {
	return s.Hub.GetRouteConfigs()
}

func (s *HttpServer) page_route_add(r *http.Request) any {
    cfg, err := s.Hub.AddRoute()
    if err != nil {
        return err
    }
    return cfg
}

func (s *HttpServer) page_route_del(r *http.Request) any {

	rid, _ := strconv.Atoi(r.FormValue("rid"))

	return s.Hub.DelRoute(scheduler.ID(rid))
}

func (s *HttpServer) page_route_config(r *http.Request) any {

	var cfg scheduler.RouteConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Hub.SetRouteConfig(cfg)
}

func (s *HttpServer) page_task_cancel(r *http.Request) any {
	gid, _ := strconv.Atoi(r.FormValue("gid"))
	tid, _ := strconv.Atoi(r.FormValue("tid"))

	return s.Hub.OrderCancel(scheduler.ID(gid), scheduler.ID(tid))
}


func httpReadJson(r *http.Request, out any) error {
	body, err := io.ReadAll(r.Body)
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

		if err, ok := rs.Data.(error); ok {
			rs.Code = 1
			rs.Data = err.Error()
		}
	}
}
