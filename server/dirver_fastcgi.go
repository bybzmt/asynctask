package server

import (
	"asynctask/server/fcgi"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
)

/*
fastcgi_split_path_info ^(.+\.php)(/?.+)$;
fastcgi_param  SCRIPT_FILENAME    $document_root$fastcgi_script_name;
fastcgi_param  PATH_INFO          $fastcgi_path_info;
fastcgi_param  PATH_TRANSLATED    $document_root$fastcgi_path_info;
fastcgi_param  QUERY_STRING       $query_string;
fastcgi_param  REQUEST_METHOD     $request_method;
fastcgi_param  CONTENT_TYPE       $content_type;
fastcgi_param  CONTENT_LENGTH     $content_length;
fastcgi_param  SCRIPT_NAME        $fastcgi_script_name;
fastcgi_param  REQUEST_URI        $request_uri;
fastcgi_param  DOCUMENT_URI       $document_uri;
fastcgi_param  DOCUMENT_ROOT      $document_root;
fastcgi_param  SERVER_PROTOCOL    $server_protocol;
fastcgi_param  HTTPS              $https if_not_empty;
fastcgi_param  GATEWAY_INTERFACE  CGI/1.1;
fastcgi_param  SERVER_SOFTWARE    nginx/$nginx_version;
fastcgi_param  REMOTE_ADDR        $remote_addr;
fastcgi_param  REMOTE_PORT        $remote_port;
fastcgi_param  SERVER_ADDR        $server_addr;
fastcgi_param  SERVER_PORT        $server_port;
fastcgi_param  SERVER_NAME        $server_name;
# PHP only, required if PHP was built with --enable-force-cgi-redirect
fastcgi_param  REDIRECT_STATUS    200;
*/

type addr struct {
	Network string
	Address string
}

type DirverFcgi struct {
	Address []string
	Params  map[string]string `json:",omitempty"`
	idx     uint32
	addrs   []addr
}

func (h *DirverFcgi) init() error {
	if len(h.Address) == 0 {
		return fmt.Errorf("DirverFcgi Address empty")
	}

	h.addrs = make([]addr, 0)

	for _, s := range h.Address {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("DirverFcgi Address empty")
		}

		_, _, err := net.SplitHostPort(s)
		if err == nil {
			h.addrs = append(h.addrs, addr{
				Network: "tcp",
				Address: s,
			})
		} else {
			h.addrs = append(h.addrs, addr{
				Network: "unix",
				Address: s,
			})
		}
	}
	return nil
}

func (h *DirverFcgi) run(o *Order) {

	if o.Task.Method == "" {
		o.Task.Method = "GET"
	}

	o.fields["url"] = o.Task.Url

	u, _ := url.Parse(o.Task.Url)

	env := map[string]string{
		"REMOTE_ADDR":     getLocalip(),
		"REQUEST_URI":     u.RequestURI(),
		"QUERY_STRING":    u.Query().Encode(),
		"REQUEST_METHOD":  o.Task.Method,
		"CONTENT_LENGTH":  strconv.Itoa(len(o.Task.Body)),
		"SCRIPT_FILENAME": u.Path,
	}

	if u.Scheme == "https" {
		env["HTTPS"] = "on"
	}

	headers := http.Header{}

	for key, val := range o.Task.Header {
		headers.Add(key, val)
	}

	host := headers.Get("Host")
	if host == "" {
		headers.Set("Host", u.Host)
	}

	for k, v := range headers {
		k = strings.Map(upperCaseAndUnderscore, k)
		if k == "PROXY" {
			// See Issue 16405
			continue
		}
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		env["HTTP_"+k] = strings.Join(v, joinStr)
	}

	var body io.Reader

	if len(o.Task.Body) > 0 {
		body = bytes.NewReader(o.Task.Body)

		bodyType := headers.Get("Content-Type")

		if bodyType == "" {
			env["CONTENT_TYPE"] = bodyType
		} else {
			env["CONTENT_TYPE"] = "application/x-www-form-urlencoded"
		}
	}

	for k, v := range h.Params {
		env[k] = v
	}

	idx := atomic.AddUint32(&h.idx, 1)

	a := h.addrs[int(idx)%len(h.addrs)]

	o.fields["fcgi"] = a.Address

	fcgi, err := fcgi.Dial(a.Network, a.Address)
	if err == nil {
		stdout, stderr, err := fcgi.Do(env, body)
		if err == nil {
			o.status, _, o.resp, o.err = readResponse(bytes.NewReader(stdout))

			if len(stderr) > 0 {
				o.fields["stderr"] = string(stderr)
			}

			if o.status == 0 {
				if len(stderr) > 0 {
					o.status = http.StatusInternalServerError
				} else {
					o.status = http.StatusOK
				}
			}

			if o.err != nil {
				return
			}

			checkHttpStatus(o)
			return
		} else {
			o.err = err
		}
	}

	o.status = -1
	o.err = err
	return
}
