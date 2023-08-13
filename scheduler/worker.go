package scheduler

import (
)

type worker struct {
	Id int
	task chan *order
	g    *group

    http workerHttp
    cli workerCli
}

func (w *worker) Init(id int, g *group) {
	w.Id = id
	w.g = g
	w.task = make(chan *order)
}

func (w *worker) Exec(t *order) {
	w.task <- t
}

func (w *worker) Cancel() {
    w.http.Cancel()
    w.cli.Cancel()
}

func (w *worker) Run() {
	for t := range w.task {
		if t == nil {
			return
		}

        if t.Task == nil {
            t.Status = -1
            t.Msg = "task is nil"
        } else if t.Task.Http != nil {
            t.Status, t.Msg = w.http.Run(t)
        } else if t.Task.Cli != nil {
            t.Status, t.Msg = w.cli.Run(t)
        } else {
            t.Status = -1
            t.Msg = "task emtpy"
        }

		w.g.complete <- t
	}
}

func (w *worker) Close() {
	close(w.task)
}
