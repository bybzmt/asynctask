package server

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestDirverCgi(t *testing.T) {
	f, err := os.CreateTemp("", "test-*.php")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())

	f.WriteString("<?php var_dump($_SERVER);")
	f.Sync()

	d := DirverCgi{
		Path: "php-cgi",
		Args: []string{"-d", "cgi.force_redirect=0"},
		Env:  []string{"SCRIPT_FILENAME=" + f.Name()},
	}

	o := Order{
		Task: Task{
			Url: "http://test/path/path2?t=1",
		},
		fields: map[string]any{},
	}

	o.ctx, o.cancel = context.WithCancel(context.Background())
	defer o.cancel()

	d.run(&o)

	t.Logf("status %d", o.status)
	t.Logf("resp %s", o.resp)

	if o.err != nil {
		t.Errorf("err: %s", o.err)
	}
}

func TestDirverCli(t *testing.T) {
	f, err := os.CreateTemp("", "test-*.php")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())

	f.WriteString("<?php var_dump($_SERVER);")
	f.Sync()

	d := DirverCgi{
		Path: "php",
		Args: []string{f.Name()},
		Cli:  true,
	}

	o := Order{
		Task: Task{
			Url: "http://test/path/path2?t=1",
		},
		fields: map[string]any{},
	}

	o.ctx, o.cancel = context.WithCancel(context.Background())
	defer o.cancel()

	d.run(&o)

	t.Logf("status %d", o.status)
	t.Logf("resp %s", o.resp)

	if o.err != nil {
		t.Errorf("err: %s", o.err)
	}
}

func TestDirverFcgi(t *testing.T) {
	f, err := os.CreateTemp("", "test-*.php")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())

	f.WriteString("<?php var_dump($_SERVER);")
	f.Sync()

	c := exec.Command("php-cgi", "-b", "127.0.0.1:8000")
	if err := c.Start(); err != nil {
		panic(err)
	}

	defer c.Process.Kill()

	time.Sleep(time.Second)

	d := DirverFcgi{
		Address: []string{"127.0.0.1:8000"},
		Params: map[string]string{
			"SCRIPT_FILENAME": f.Name(),
		},
	}
	d.init()

	t.Logf("fcgi %#v", d)

	o := Order{
		Task: Task{
			Url: "http://test/path/path2?t=1",
		},
		fields: map[string]any{},
	}

	o.ctx, o.cancel = context.WithCancel(context.Background())
	defer o.cancel()

	d.run(&o)

	t.Logf("status %d", o.status)
	t.Logf("resp %s", o.resp)
	t.Logf("resp %#v", o.fields)

	if o.err != nil {
		t.Errorf("err: %s", o.err)
	}
}
