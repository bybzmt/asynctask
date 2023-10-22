package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

var ErrHttpStatus = errors.New("Code != 200")

type workerHttp struct {
	order  *order
	req    *http.Request
	cancel context.CancelFunc
}

func (w *workerHttp) init() error {

	var uri string

	if w.order.Base.HttpBase != "" {
		uri = w.order.Base.HttpBase + w.order.Task.Url
	} else {
		uri = w.order.Task.Url
	}

	var err error

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	if u.Hostname() == "" {
		return errors.New("host is empty")
	}

	header := url.Values{}

	for k, v := range w.order.Base.HttpHeader {
		header.Set(k, v)
	}

	method := w.order.Task.Method
	var body []byte

	if w.order.Task.Body != nil {
		if method == "" {
			method = "POST"
		}

		var t string

		if err := json.Unmarshal(w.order.Task.Body, &t); err != nil {
			body = w.order.Task.Body

			if !header.Has("Content-Type") {
				header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		} else {
			if !header.Has("Content-Type") {
				header.Set("Content-Type", "application/json")
			}

			body = []byte(t)
		}
	} else if w.order.Task.Form != nil {

		if method == "GET" {
			q := u.Query()

			for k, v := range w.order.Task.Form {
				q.Set(k, v)
			}

			u.RawQuery = q.Encode()
		} else {
			if method == "" {
				method = "POST"
			}

			if !header.Has("Content-Type") {
				header.Set("Content-Type", "application/x-www-form-urlencoded")
			}

			t := url.Values{}

			for k, v := range w.order.Task.Form {
				t.Set(k, v)
			}

			body = []byte(t.Encode())
		}
	}

	timeout := w.order.Task.Timeout

	if w.order.Base.Timeout > 0 {
		if timeout < 1 || timeout > w.order.Base.Timeout {
			timeout = w.order.Base.Timeout
		}
	}
	if timeout < 1 {
		timeout = 60
	}

	ctx, cancel := context.WithTimeout(w.order.ctx, time.Duration(timeout)*time.Second)
	w.cancel = cancel

	var rb io.Reader
	if body != nil {
		rb = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), rb)
	if err != nil {
		return err
	}

	w.req = req

	return nil
}

func (w *workerHttp) run() (status int, msg string) {

	if err := w.init(); err != nil {
		status = -1
		w.order.Err = err
		return
	}

    w.order.taskTxt = w.req.URL.String()

    defer w.cancel()

    resp, err := w.order.job.s.Client.Do(w.req)
	if err != nil {
		status = -1
		w.order.Err = err
		return
	}

	defer resp.Body.Close()

	b2, _ := io.ReadAll(resp.Body)

	status = resp.StatusCode
	msg = string(b2)

	if w.order.Task.Code == 0 {
		if !(status >= 200 && status < 300) {
            w.order.Err = ErrHttpStatus
		}
	} else if w.order.Task.Code != status {
        w.order.Err = ErrHttpStatus
	}

	return
}
