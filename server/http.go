package server

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

func (s *Server) initHttp() {
	h := http.NewServeMux()

	tfs, _ := fs.Sub(uifiles, "dist")
	h.Handle("/", http.FileServer(http.FS(tfs)))

	h.HandleFunc("/api/task/status", page_error(s.page_status))
	h.HandleFunc("/api/group/status", page_error(s.page_groups_status))

	h.HandleFunc("/api/task/runing", page_error(s.page_runing))
	h.HandleFunc("/api/task/add", page_error(s.page_task_add))
	h.HandleFunc("/api/task/cancel", page_error(s.page_task_cancel))
	h.HandleFunc("/api/job/delIdle", page_error(s.page_job_delIdle))
	h.HandleFunc("/api/job/emptyAll", page_error(s.page_job_empty))
	h.HandleFunc("/api/job/setConfig", page_error(s.page_job_config))
	h.HandleFunc("/api/routes", page_error(s.page_routes))
	h.HandleFunc("/api/route/add", page_error(s.page_route_add))
	h.HandleFunc("/api/route/del", page_error(s.page_route_del))
	h.HandleFunc("/api/route/setConfig", page_error(s.page_route_config))
	h.HandleFunc("/api/group/add", page_error(s.page_group_add))
	h.HandleFunc("/api/group/del", page_error(s.page_group_del))
	h.HandleFunc("/api/group/setConfig", page_error(s.page_group_config))

	s.Http.Handler = h
}

func (s *Server) page_status(r *http.Request) any {
	return s.Scheduler.GetStatData()
}

func (s *Server) page_runing(r *http.Request) any {
	return s.Scheduler.GetRunTaskStat()
}

func (s *Server) page_task_add(r *http.Request) any {
	var o scheduler.Task

	if err := httpReadJson(r, &o); err != nil {
		return err
	}

	return s.Scheduler.AddTask(&o)
}

func (s *Server) page_job_empty(r *http.Request) any {
	name := r.FormValue("name")

	return s.Scheduler.JobEmpty(name)
}

func (s *Server) page_job_delIdle(r *http.Request) any {
	jname := strings.TrimSpace(r.FormValue("name"))

	return s.Scheduler.JobDelIdle(jname)
}

func (s *Server) page_job_config(r *http.Request) any {
	jname := strings.TrimSpace(r.FormValue("name"))
	_cfg := r.FormValue("cfg")

	var cfg scheduler.JobConfig

	if err := json.Unmarshal([]byte(_cfg), &cfg); err != nil {
		return err
	}

	return s.Scheduler.SetJobConfig(jname, cfg)
}

func (s *Server) page_groups_status(r *http.Request) any {
	return s.Scheduler.GetGroupStat()
}

func (s *Server) page_group_add(r *http.Request) any {
	cfg, err := s.Scheduler.AddGroup()
	if err != nil {
		return err
	}
	return cfg
}

func (s *Server) page_group_del(r *http.Request) any {

	gid, _ := strconv.Atoi(r.FormValue("gid"))

	return s.Scheduler.DelGroup(scheduler.ID(gid))
}

func (s *Server) page_group_config(r *http.Request) any {

	var cfg scheduler.GroupConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.SetGroupConfig(cfg)
}

func (s *Server) page_routes(r *http.Request) any {
	return s.Scheduler.GetRouteConfigs()
}

func (s *Server) page_route_add(r *http.Request) any {
	cfg, err := s.Scheduler.AddRoute()
	if err != nil {
		return err
	}
	return cfg
}

func (s *Server) page_route_del(r *http.Request) any {

	rid, _ := strconv.Atoi(r.FormValue("rid"))

	return s.Scheduler.DelRoute(scheduler.ID(rid))
}

func (s *Server) page_route_config(r *http.Request) any {

	var cfg scheduler.TaskConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.SetRouteConfig(cfg)
}

func (s *Server) page_task_cancel(r *http.Request) any {
	gid, _ := strconv.Atoi(r.FormValue("gid"))
	tid, _ := strconv.Atoi(r.FormValue("tid"))

	return s.Scheduler.OrderCancel(scheduler.ID(gid), scheduler.ID(tid))
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
