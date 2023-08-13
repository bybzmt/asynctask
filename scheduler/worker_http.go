package scheduler

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

var ErrHttpStatus = errors.New("Code != 200")

type workerHttp struct {
	resp atomic.Value
}

func (w *workerHttp) Cancel() {
	_resp := w.resp.Load()

	if _resp != nil {
		fn := _resp.(context.CancelFunc)
		if fn != nil {
			fn()
		}
        w.resp.Store(nil)
	}
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
		timeout = 60 * 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

	w.resp.Store(cancel)
    defer w.resp.Store(nil)

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		status = -1
		msg = err.Error()
		o.Err = err
		return
	}

	resp, err = o.job.g.s.Client.Do(req)
	if err != nil {
		status = -1
		msg = err.Error()
		o.Err = err
		return
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(body)

	if status != 200 {
		o.Err = ErrHttpStatus
	}
	return
}
