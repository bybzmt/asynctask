package server

import (
	"testing"
)

func TestRouter(t *testing.T) {

	rs := []*Route{
		{
			Pattern: "^http://test1/(.+)",
			Job:     "slow",
			Rewrite: &Rewrite{
				Method:  "POST",
				Pattern: "/test1/",
				Rewrite: "/127.0.0.1/",
			},
		},
		{
			Pattern: "test2/(.+)",
			Job:     "test2/$1",
			Rewrite: &Rewrite{
				Header: map[string]string{
					"Host": "host_new",
				},
				Pattern: "^test2/",
				Rewrite: "http://127.0.0.1/",
			},
		},
	}

	r := router{}
	r.set(rs)

	t1 := Task{
		Url: "http://test1/123?123",
		Header: map[string]string{
			"Host": "host_old",
		},
	}

	if o, err := r.route(&t1); err == nil {
		t.Log("task1", json_encode(o))

		if o.Job != "slow" {
			t.Error("route1 Job not Slow")
		}

		if o.Task.Url != "http://127.0.0.1/123?123" {
			t.Error("route1 url error")
		}

		if o.Task.Method != "POST" {
			t.Error("route1 Method error")
		}
	} else {
		t.Error("route1", err)
	}

	t2 := Task{
		Url: "test2/123?123",
		Header: map[string]string{
			"Host": "host_old",
		},
	}

	if o, err := r.route(&t2); err == nil {
		t.Log("task2", json_encode(o))

		if o.Task.Header["Host"] != "host_new" {
			t.Error("route2 Header Host error")
		}
	} else {
		t.Error("route2:", err)
	}

	t3 := Task{
		Url: "testNotAollow/123?123",
	}

	if _, err := r.route(&t3); err == nil {
		t.Error("task3 mast not allow")
	}

	t4 := Task{
		Url: "http://test1/123?123",
	}
	if o, err := r.route(&t4); err == nil {
		t.Log("task4", json_encode(o))

		if o.Task.Header["Host"] != "" {
			t.Error("Task4 Header Host error")
		}
	}

}
