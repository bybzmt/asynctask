package server

import (
	"asynctask/scheduler"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"runtime"
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
	h.HandleFunc("/api/task/del", page_error(s.page_task_del))
	h.HandleFunc("/api/task/check", page_error(s.page_task_check))
	h.HandleFunc("/api/task/cancel", page_error(s.page_task_cancel))
	h.HandleFunc("/api/task/timed", page_error(s.page_task_timed))
	h.HandleFunc("/api/task/empty", page_error(s.page_task_empty))
	h.HandleFunc("/api/task/delIdle", page_error(s.page_job_delIdle))
	h.HandleFunc("/api/config", page_error(s.page_config))

	c := &http.Server{
		Addr:    s.cfg.HttpAddr,
		Handler: h,
	}

	s.http = c
}

type Stat struct {
	scheduler.Stat
	Timed int
}

func (s *Server) page_status(r *http.Request) any {
	t := &Stat{
		Stat:  s.s.GetStat(),
		Timed: s.timer.Len(),
	}
	return t
}

func (s *Server) page_runing(r *http.Request) any {
	var out []any

	ts := s.s.GetRunTask()

	for _, t := range ts {
		o := s.store_order_get(t.Id)

		out = append(out, struct {
			scheduler.RunTask
			Task Task
		}{
			Task:    o.Task,
			RunTask: t,
		})
	}

	return out
}

func (s *Server) page_task_add(r *http.Request) any {
	var t Task

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.TaskAdd(&t)
}

func (s *Server) page_task_del(r *http.Request) any {
	var cfg struct {
		Id ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.TaskDel(cfg.Id)
}

func (s *Server) page_task_check(r *http.Request) any {
	var t Task

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	o, err := s.TaskCheck(&t)
	if err != nil {
		return err
	}

	return o
}

func (s *Server) page_task_empty(r *http.Request) any {
	var t struct {
		Name string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.TaskEmpty(t.Name)
}

func (s *Server) page_job_delIdle(r *http.Request) any {
	var t struct {
		Name string
	}

	if err := httpReadJson(r, &t); err != nil {
		return err
	}

	return s.JobDel(t.Name)
}

func (s *Server) page_task_cancel(r *http.Request) any {
	var cfg struct {
		Id ID
	}

	if err := httpReadJson(r, &cfg); err != nil {
		return err
	}

	return s.TaskCancel(cfg.Id)
}

func (s *Server) page_task_timed(r *http.Request) any {
	return s.TimerTopN(100)
}

func (s *Server) page_config(r *http.Request) any {
	s.l.Lock()
	defer s.l.Unlock()

	return s.cfg
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
				buf := make([]byte, 4096)

				n := runtime.Stack(buf, false)

				rs.Code = 1
				rs.Data = struct {
					Err   any
					Stack string
				}{
					Err:   err,
					Stack: string(buf[0:n]),
				}
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
