package scheduler

import (
)

type worker struct {
	Id ID
	task chan *order
	g    *group

    http workerHttp
    cli workerCli
}

func (w *worker) Init(id ID, g *group) {
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
    w.g.s.Log.Debugln("workder run", w.Id)

	for t := range w.task {
		if t == nil {
			return
		}

        if t.Base.Mode & MODE_HTTP == MODE_HTTP {
            if t.Task.Http == nil {
                t.Status = -1
                t.Msg = "task http is nil"
            } else {
                t.Status, t.Msg = w.http.Run(t)
            }
        } else if t.Base.Mode & MODE_CMD == MODE_CMD {
            if t.Task.Cli == nil {
                t.Status = -1
                t.Msg = "task cli is nil"
            } else {
                t.Status, t.Msg = w.cli.Run(t)
            }
        }

		w.g.complete <- t
	}
}

func (w *worker) Close() {
	close(w.task)
}
