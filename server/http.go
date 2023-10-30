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
	h.HandleFunc("/api/task/runing", page_error(s.page_runing))
	h.HandleFunc("/api/task/add", page_error(s.page_task_add))
	h.HandleFunc("/api/task/cancel", page_error(s.page_task_cancel))
	h.HandleFunc("/api/task/timed", page_error(s.page_task_timed))
	h.HandleFunc("/api/task/timeddel", page_error(s.page_task_timed_del))
	h.HandleFunc("/api/task/empty", page_error(s.page_task_empty))
	h.HandleFunc("/api/task/delIdle", page_error(s.page_job_delIdle))

	h.HandleFunc("/api/router/get", page_error(s.page_router_get))
	h.HandleFunc("/api/router/set", page_error(s.page_router_set))

	h.HandleFunc("/api/rule/list", page_error(s.page_rules))
	h.HandleFunc("/api/rule/put", page_error(s.page_rule_put))
	h.HandleFunc("/api/rule/del", page_error(s.page_rule_del))

	h.HandleFunc("/api/group/list", page_error(s.page_groups))
	h.HandleFunc("/api/group/add", page_error(s.page_group_add))
	h.HandleFunc("/api/group/del", page_error(s.page_group_del))
	h.HandleFunc("/api/group/set", page_error(s.page_group_config))

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

	return s.Scheduler.TaskAdd(o)
}

func (s *Server) page_task_empty(r *http.Request) any {
	var t struct {
		Name string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.TaskEmpty(t.Name)
}

func (s *Server) page_job_delIdle(r *http.Request) any {
	var t struct {
		Name string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.TaskDelIdle(t.Name)
}

func (s *Server) page_groups(r *http.Request) any {
	return s.Scheduler.Groups()
}

func (s *Server) page_group_add(r *http.Request) any {

	var c scheduler.GroupConfig

	if err := httpReadJson(r, &c); err != nil {
		return err
	}

	return s.Scheduler.GroupAdd(c)
}

func (s *Server) page_group_del(r *http.Request) any {
	var cfg struct {
		Id scheduler.ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.GroupDel(cfg.Id)
}

func (s *Server) page_group_config(r *http.Request) any {

	var cfg scheduler.GroupConfig

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.GroupConfig(cfg)
}

func (s *Server) page_router_set(r *http.Request) any {

	var cfg []string

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.SetRoutes(cfg)
}

func (s *Server) page_router_get(r *http.Request) any {
	return s.Scheduler.Routes()
}

func (s *Server) page_task_cancel(r *http.Request) any {
	var cfg struct {
		Id scheduler.ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.Scheduler.TaskCancel(cfg.Id)
}

func (s *Server) page_task_timed(r *http.Request) any {
	var t struct {
		Starttime int
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.TimerShow(t.Starttime, 100)
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

func (s *Server) page_rules(r *http.Request) any {
	return s.Scheduler.Rules()
}

func (s *Server) page_rule_put(r *http.Request) any {
	var t scheduler.Rule

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.RulePut(t)
}

func (s *Server) page_rule_del(r *http.Request) any {
	var t struct {
		Type    scheduler.RuleType
		Pattern string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.Scheduler.RuleDel(t.Type, t.Pattern)
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
