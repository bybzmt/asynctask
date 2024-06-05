package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/http/httpguts"
)

type Dirver struct {
	Type DirverType
	Cgi  *DirverCgi  `json:",omitempty"`
	Fcgi *DirverFcgi `json:",omitempty"`
	http *dirverHttp
}

func (d *Dirver) run(o *Order) {
	switch d.Type {
	case DIRVER_HTTP:
		d.http.run(o)

	case DIRVER_CGI:
		d.Cgi.run(o)

	case DIRVER_FASTCGI:
		d.Fcgi.run(o)

	default:
		o.err = TaskError
	}
}

func (d *Dirver) init(s *Server) error {

	if d.Type == 0 {
		if d.Cgi == nil && d.Fcgi != nil {
			d.Type = DIRVER_FASTCGI
		} else if d.Cgi != nil && d.Fcgi == nil {
			d.Type = DIRVER_CGI
		}
	}

	switch d.Type {
	case DIRVER_HTTP:
		d.http = &dirverHttp{
			client: &http.Client{},
		}

	case DIRVER_CGI:
		if d.Cgi == nil {
			return fmt.Errorf("Cgi is nil")
		}

		if d.Cgi.Path == "" {
			return fmt.Errorf("Cgi.Path empty")
		}

	case DIRVER_FASTCGI:
		if d.Fcgi == nil {
			return fmt.Errorf("Fcgi is nil")
		}

		return d.Fcgi.init()

	default:
		return fmt.Errorf("Dirver Type Unknow")
	}

	return nil
}

type dirverHttp struct {
	client *http.Client
}

func (d *dirverHttp) run(o *Order) {

	t := o.Task

	var rb io.Reader
	if t.Body != nil {
		rb = bytes.NewReader(t.Body)
	}

	req, err := http.NewRequestWithContext(o.ctx, t.Method, t.Url, rb)
	if err != nil {
		o.err = err
		return
	}

	resp, err := d.client.Do(req)

	defer func() {
		if o.err != nil {
			o.resp = append(o.resp, "\n==request==\n"...)

			var buf bytes.Buffer
			err := req.Write(&buf)
			if err != nil {
				o.resp = append(o.resp, err.Error()...)
			} else {
				o.resp = append(o.resp, buf.Bytes()...)
			}
		}
	}()

	if err != nil {
		o.err = err
		return
	}

	onResponse(o, resp)
}

func onResponse(o *Order, resp *http.Response) {
	defer resp.Body.Close()

	b2, _ := io.ReadAll(resp.Body)

	o.status = resp.StatusCode
	o.resp = b2

	checkHttpStatus(o)
}

func upperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		// Maybe not part of the CGI 'spec' but would mess up
		// the environment in any case, as Go represents the
		// environment as a slice of "key=value" strings.
		return '_'
	}
	// TODO: other transformations in spec or practice?
	return r
}

func checkHttpStatus(o *Order) {
	if o.Task.Status == 0 {
		if !(o.status >= 200 && o.status < 300) {
			o.err = fmt.Errorf("Status %d", o.status)
		}
	} else if o.Task.Status != o.status {
		o.err = fmt.Errorf("Status %d != %d", o.status, o.Task.Status)
	}
}

func readResponse(r io.Reader) (status int, headers http.Header, body []byte, err error) {

	linebody := bufio.NewReaderSize(r, 1024)
	headers = make(http.Header)
	statusCode := 0
	headerLines := 0
	sawBlankLine := false
	for {
		line, isPrefix, err2 := linebody.ReadLine()

		if isPrefix {
			status = http.StatusInternalServerError
			err = fmt.Errorf("cgi: long header line from subprocess.")
			return
		}
		if err2 == io.EOF {
			break
		}
		if err2 != nil {
			err = fmt.Errorf("cgi: error reading headers: %v", err2)
			return
		}
		if len(line) == 0 {
			sawBlankLine = true
			break
		}
		headerLines++
		header, val, ok := strings.Cut(string(line), ":")
		if !ok {
			status = http.StatusInternalServerError
			err = fmt.Errorf("cgi: bogus header line: %s\n", string(line))
			return
		}
		if !httpguts.ValidHeaderFieldName(header) {
			status = http.StatusInternalServerError
			err = fmt.Errorf("cgi: invalid header name: %q\n", header)
			return
		}
		val = textproto.TrimString(val)
		switch {
		case header == "Status":
			if len(val) < 3 {
				status = http.StatusInternalServerError
				err = fmt.Errorf("cgi: bogus status (short): %q\n", val)
				return
			}
			code, err2 := strconv.Atoi(val[0:3])
			if err2 != nil {
				status = http.StatusInternalServerError
				err = fmt.Errorf("cgi: bogus status: %q line was %q\n", val, line)
				return
			}
			statusCode = code
		default:
			headers.Add(header, val)
		}
	}

	if headerLines == 0 || !sawBlankLine {
		status = http.StatusInternalServerError
		err = fmt.Errorf("cgi: no headers")
		return
	}

	if loc := headers.Get("Location"); loc != "" {
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	}

	if statusCode == 0 && headers.Get("Content-Type") == "" {
		status = http.StatusInternalServerError
		err = fmt.Errorf("cgi: missing required Content-Type in headers")
		return
	}

	status = statusCode
	body, err = io.ReadAll(linebody)
	return
}

var ip string
var hostname string

func getLocalip() string {
	if ip == "" {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			ip = "0.0.0.0"
		}

		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	return ip
}

func getHostname() string {
	if hostname == "" {
		name, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		} else {
			hostname = name
		}
	}

	return hostname
}
