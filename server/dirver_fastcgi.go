package server

import (
	"bytes"
	"net/url"
	"strconv"

	"github.com/tomasen/fcgi_client"
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

type DirverFcgi struct {
	Network string
	Address string
	Params  map[string]string
}

func (h *DirverFcgi) run(o *Order) {

	u, _ := url.Parse(o.Task.Url)

	env := make(map[string]string)
	for k, v := range h.Params {
		env[k] = v
	}

	env["REMOTE_ADDR"] = "127.0.0.1"
	env["REQUEST_URI"] = u.Path
	env["QUERY_STRING"] = u.Query().Encode()
    env["REQUEST_METHOD"] = o.Task.Method
	env["CONTENT_LENGTH"] = strconv.Itoa(len(o.Task.Body))

	if len(o.Task.Body) > 0 {
        bodyType, ok := o.Task.Header["ContentType"]

        if ok  {
            env["CONTENT_TYPE"] = bodyType
        } else {
            env["CONTENT_TYPE"] = "application/x-www-form-urlencoded"
        }
    }

	fcgi, err := fcgiclient.Dial(h.Network, h.Address)
	if err == nil {
        r := bytes.NewReader(o.Task.Body)

		resp, err := fcgi.Request(env, r)
		if err == nil {
			onResponse(o, resp)
			return
		}
	}

	o.status = -1
	o.err = err
	return
}
