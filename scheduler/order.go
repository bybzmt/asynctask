package scheduler

import (
	"context"
	"time"
)

type order struct {
	id ID

	job *job
	g   *group

	dirver Dirver
	log    Logger

	ctx    context.Context
	cancel context.CancelFunc

	err error

	startTime time.Time
	statTime  time.Time
}

func (o *order) run() {
	o.log.Println("task run", o.id)

	o.err = o.dirver.Run(o.id, o.ctx)

	if o.err != nil {
		o.log.Println("task err", o.id, o.err.Error())
	} else {
		o.log.Println("task end", o.id)
	}

	o.g.s.complete <- o
}
