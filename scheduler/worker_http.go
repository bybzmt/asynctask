package scheduler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var ErrHttpStatus = errors.New("Code != 200")

type workerHttp struct {
	l    sync.Mutex
	resp context.CancelFunc
}

func (w *workerHttp) Cancel() {
	w.l.Lock()
	defer w.l.Unlock()

	if w.resp != nil {
		w.resp()
		w.resp = nil
	}
}

var nop = func() {
}

func (w *workerHttp) Run(o *order) (status int, msg string) {
	var uri string

	if o.Base.HttpBase != "" {
		uri = o.Base.HttpBase + o.Task.Http.Url
	} else {
		uri = o.Task.Http.Url
	}

	var resp *http.Response
	var err error

	u, err := url.Parse(uri)
	if err != nil {
		status = -1
		msg = err.Error()
		o.Err = err
		return
	}

	if u.Hostname() == "" {
		status = -1
		msg = "host is empty"
		return
	}

	q := u.Query()

	for k, v := range o.Task.Http.Get {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()

	timeout := o.Task.Timeout

	if o.Base.Timeout > 0 {
		if timeout < 1 || timeout > o.Base.Timeout {
			timeout = o.Base.Timeout
		}
	}
	if timeout < 1 {
		timeout = 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

	w.l.Lock()
	w.resp = cancel
	w.l.Unlock()

	defer func() {
		w.l.Lock()
		defer w.l.Unlock()
		w.resp = nil
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		status = -1
		msg = err.Error()
		o.Err = err
		return
	}

    for k, v := range o.Base.HttpHeader {
        req.Header.Set(k, v)
    }

	resp, err = o.job.g.s.Client.Do(req)
	if err != nil {
		status = -1
		msg = err.Error()
		o.Err = err
		return
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(body)

	if !(status >= 200 && status < 300) {
		o.Err = ErrHttpStatus
	}

	return
}
