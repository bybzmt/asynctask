package server

import (
	"asynctask/scheduler"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
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
	h.HandleFunc("/api/task/timed", page_error(s.page_task_timed))
	h.HandleFunc("/api/task/timeddel", page_error(s.page_task_timed_del))

	h.HandleFunc("/api/job/delIdle", page_error(s.page_job_delIdle))
	h.HandleFunc("/api/job/empty", page_error(s.page_job_empty))
	h.HandleFunc("/api/job/setConfig", page_error(s.page_job_config))

	h.HandleFunc("/api/routes", page_error(s.page_routes))
	h.HandleFunc("/api/route/add", page_error(s.page_route_add))
	h.HandleFunc("/api/route/del", page_error(s.page_route_del))
	h.HandleFunc("/api/route/setConfig", page_error(s.page_route_config))

	h.HandleFunc("/api/group/add", page_error(s.page_group_add))
	h.HandleFunc("/api/group/del", page_error(s.page_group_del))
	h.HandleFunc("/api/group/setConfig", page_error(s.page_group_config))

	h.HandleFunc("/api/cron/getConfig", page_error(s.page_cron_getConfig))
	h.HandleFunc("/api/cron/setConfig", page_error(s.page_cron_setConfig))
	h.HandleFunc("/api/cron/reload", page_error(s.page_cron_reload))

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

	return s.Scheduler.TaskAdd(&o)
}

func (s *Server) page_job_empty(r *http.Request) any {
	var t struct {
		Name string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.JobEmpty(t.Name)
}

func (s *Server) page_job_delIdle(r *http.Request) any {
	var t struct {
		Name string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.JobDelIdle(t.Name)
}

func (s *Server) page_job_config(r *http.Request) any {
	var cfg struct {
		scheduler.JobConfig
		Name string
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.SetJobConfig(cfg.Name, cfg.JobConfig)
}

func (s *Server) page_groups_status(r *http.Request) any {
	return s.Scheduler.GetGroupStat()
}

func (s *Server) page_group_add(r *http.Request) any {

	var t struct{}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	cfg, err := s.Scheduler.AddGroup()
	if err != nil {
		return err
	}
	return cfg
}

func (s *Server) page_group_del(r *http.Request) any {
	var cfg struct {
		gid scheduler.ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.DelGroup(cfg.gid)
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
	var cfg struct {
		rid scheduler.ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.DelRoute(cfg.rid)
}

func (s *Server) page_route_config(r *http.Request) any {

	var cfg scheduler.TaskConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.SetRouteConfig(cfg)
}

func (s *Server) page_task_cancel(r *http.Request) any {
	var cfg struct {
		tid scheduler.ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.TaskCancel(cfg.tid)
}

func (s *Server) page_task_timed(r *http.Request) any {
	var t struct {
		starttime int
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.TimerShow(t.starttime, 100)
}

func (s *Server) page_task_timed_del(r *http.Request) any {
	var t struct {
		TimedID string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.TimerDel(t.TimedID)
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
