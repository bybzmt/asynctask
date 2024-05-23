package server

import (
	"testing"
)

func TestRouter(t *testing.T) {

	rs := []*Route{
		{
			Pattern: "^regular/priority/(.+)",
			Job:     "job/$1",
			Rewrite: &Rewrite{
				Method: "POST",
				Header: map[string]string{
					"Host": "host2",
				},
				Pattern: "^regular/priority/(.+)",
				Rewrite: "rewrite/$1",
			},
		},
		{
			Pattern: "^regular/(.+)",
		},
		{
			Pattern: "regular/map1",
			Job:     "hasjob",
		},
		{
			Pattern: "regular/map2",
		},
	}

	for _, r := range rs {
		t.Log("route", json_encode(r))
	}

	r := router{}
	r.set(rs)

	if o, err := r.route(&Task{
		Url: "regular/priority/123?a=1",
		Header: map[string]string{
			"Host": "host1",
		},
	}); err == nil {
		a1 := `{"Job":"job/123","Task":{"method":"POST","url":"rewrite/123?a=1","header":{"Host":"host2"}}}`
		a2 := json_encode(o)

		t.Log("route expect", a1)

		if a1 != a2 {
			t.Error("route fail", a2)
		}
	} else {
		t.Error("route fail", err)
	}

	if o, err := r.route(&Task{Url: "regular/test2?a=1"}); err == nil {
		a1 := `{"Job":"regular/test2","Task":{"url":"regular/test2?a=1"}}`
		a2 := json_encode(o)

		t.Log("route expect", a1)

		if a1 != a2 {
			t.Error("route fail", a2)
		}
	} else {
		t.Error("route fail", err)
	}

	if o, err := r.route(&Task{Url: "regular/map1?a=1"}); err == nil {
		a1 := `{"Job":"hasjob","Task":{"url":"regular/map1?a=1"}}`
		a2 := json_encode(o)

		t.Log("route expect", a1)

		if a1 != a2 {
			t.Error("route fail", a2)
		}
	} else {
		t.Error("route fail", err)
	}

	if o, err := r.route(&Task{Url: "regular/map2?a=1"}); err == nil {
		a1 := `{"Job":"regular/map2","Task":{"url":"regular/map2?a=1"}}`
		a2 := json_encode(o)

		t.Log("route expect", a1)

		if a1 != a2 {
			t.Error("route fail", a2)
		}
	} else {
		t.Error("route fail", err)
	}

	if _, err := r.route(&Task{Url: "testNotAollow"}); err == nil {
		t.Error("route mast not allow")
	}

}
