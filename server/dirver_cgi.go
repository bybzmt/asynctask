package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var trailingPort = regexp.MustCompile(`:([0-9]+)$`)

var osDefaultInheritEnv = func() []string {
	switch runtime.GOOS {
	case "darwin", "ios":
		return []string{"DYLD_LIBRARY_PATH"}
	case "linux", "freebsd", "netbsd", "openbsd":
		return []string{"LD_LIBRARY_PATH"}
	case "hpux":
		return []string{"LD_LIBRARY_PATH", "SHLIB_PATH"}
	case "irix":
		return []string{"LD_LIBRARY_PATH", "LD_LIBRARYN32_PATH", "LD_LIBRARY64_PATH"}
	case "illumos", "solaris":
		return []string{"LD_LIBRARY_PATH", "LD_LIBRARY_PATH_32", "LD_LIBRARY_PATH_64"}
	case "windows":
		return []string{"SystemRoot", "COMSPEC", "PATHEXT", "WINDIR"}
	}
	return nil
}()

type DirverCgi struct {
	Path string `json:",omitempty"` // path to the CGI executable
	Root string `json:",omitempty"` // root URI prefix of handler or empty for "/"

	// Dir specifies the CGI executable's working directory.
	// If Dir is empty, the base directory of Path is used.
	// If Path has no base directory, the current working
	// directory is used.
	Dir        string   `json:",omitempty"`
	Args       []string `json:",omitempty"` // optional arguments to pass to child process
	Env        []string `json:",omitempty"` // extra environment variables to set, if any, as "key=value"
	InheritEnv []string `json:",omitempty"` // environment variables to inherit from host, as "key"
	Cli        bool     `json:",omitempty"`
}

func (h *DirverCgi) init() error {
	if h.Path == "" {
		return fmt.Errorf("DirverCgi Path empty")
	}

	return nil
}

// removeLeadingDuplicates remove leading duplicate in environments.
// It's possible to override environment like following.
//
//	cgi.cgiHandler{
//	  ...
//	  Env: []string{"SCRIPT_FILENAME=foo.php"},
//	}
func removeLeadingDuplicates(env []string) (ret []string) {
	for i, e := range env {
		found := false
		if eq := strings.IndexByte(e, '='); eq != -1 {
			keq := e[:eq+1] // "key="
			for _, e2 := range env[i+1:] {
				if strings.HasPrefix(e2, keq) {
					found = true
					break
				}
			}
		}
		if !found {
			ret = append(ret, e)
		}
	}
	return
}

func (h *DirverCgi) run(o *Order) {

	req := new(http.Request).WithContext(o.ctx)

	u, _ := url.Parse(o.Task.Url)

	req.Method = o.Task.Method
	req.URL = u
	req.Header = make(http.Header)

	if req.Method == "" {
		req.Method = "GET"
	}

	for k, v := range o.Task.Header {
		req.Header.Set(k, v)
	}

	req.Host = req.Header.Get("Host")
	if req.Host == "" {
		req.Host = u.Host
	}

	req.Body = io.NopCloser(bytes.NewReader(o.Task.Body))

	h.ServeHTTP(req, o)
}

func (h *DirverCgi) ServeHTTP(req *http.Request, o *Order) {
	root := h.Root
	if root == "" {
		root = "/"
	}

	if len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked" {
		o.status = http.StatusBadRequest
		o.err = fmt.Errorf("Chunked request bodies are not supported by CGI.")
		return
	}

	pathInfo := req.URL.Path
	if root != "/" && strings.HasPrefix(pathInfo, root) {
		pathInfo = pathInfo[len(root):]
	}

	env := []string{
		"HTTP_HOST=" + req.Host,
		"REQUEST_METHOD=" + req.Method,
		"QUERY_STRING=" + req.URL.RawQuery,
		"REQUEST_URI=" + req.URL.RequestURI(),
		"PATH_INFO=" + pathInfo,
		"SCRIPT_NAME=" + root,
		"SCRIPT_FILENAME=" + h.Path,
		"REMOTE_ADDR=" + getLocalip(),
	}

	if hostDomain, _, err := net.SplitHostPort(req.Host); err == nil {
		env = append(env, "SERVER_NAME="+hostDomain)
	} else {
		env = append(env, "SERVER_NAME="+req.Host)
	}

	if req.URL.Scheme == "https" {
		env = append(env, "HTTPS=on")
	}

	for k, v := range req.Header {
		k = strings.Map(upperCaseAndUnderscore, k)
		if k == "PROXY" {
			// See Issue 16405
			continue
		}
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		env = append(env, "HTTP_"+k+"="+strings.Join(v, joinStr))
	}

	if req.ContentLength > 0 {
		env = append(env, fmt.Sprintf("CONTENT_LENGTH=%d", req.ContentLength))
	}
	if ctype := req.Header.Get("Content-Type"); ctype != "" {
		env = append(env, "CONTENT_TYPE="+ctype)
	}

	envPath := os.Getenv("PATH")
	if envPath == "" {
		envPath = "/bin:/usr/bin:/usr/ucb:/usr/bsd:/usr/local/bin"
	}
	env = append(env, "PATH="+envPath)

	for _, e := range h.InheritEnv {
		if v := os.Getenv(e); v != "" {
			env = append(env, e+"="+v)
		}
	}

	for _, e := range osDefaultInheritEnv {
		if v := os.Getenv(e); v != "" {
			env = append(env, e+"="+v)
		}
	}

	if h.Env != nil {
		env = append(env, h.Env...)
	}

	env = removeLeadingDuplicates(env)

	cmd := exec.CommandContext(req.Context(), h.Path, h.Args...)
	cmd.Env = env

	if h.Dir != "" {
		cmd.Dir = h.Dir
	}

	if req.ContentLength != 0 {
		cmd.Stdin = req.Body
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	o.fields["url"] = req.URL.String()

	err := cmd.Run()
	if err != nil {
		o.status = http.StatusInternalServerError
		o.err = err
		o.fields["cmd"] = cmd.String()
		o.fields["env"] = strings.Join(env, "; ")
		o.fields["stdout"] = out.String()
		o.fields["stderr"] = stderr.String()
		return
	}

	if t := stderr.Bytes(); len(t) > 0 {
		o.fields["stderr"] = stderr.String()
	}

	if h.Cli {
		o.status = cmd.ProcessState.ExitCode()
		o.resp = out.Bytes()

		if o.Task.Status != o.status {
			o.err = fmt.Errorf("Status %d != %d", o.status, o.Task.Status)
		}

		return
	}

	o.status, _, o.resp, o.err = readResponse(&out)

	if o.err != nil {
		o.fields["cmd"] = cmd.String()
		o.fields["env"] = strings.Join(env, "; ")
		return
	}

	if o.status == 0 {
		if len(stderr.Bytes()) > 0 {
			o.status = http.StatusInternalServerError
		} else {
			o.status = http.StatusOK
		}
	}

	checkHttpStatus(o)
}
