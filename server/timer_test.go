package server

import (
	"testing"
)

func TestTimer(t *testing.T) {

	ts := timer{}
	ts.init()

    max := 10000

	for i := 0; i < 30; i++ {
		id := ts_getRand() % max
		ts.push(int64(id), ID(id))
	}

	var old ID = 0

	for ts.num > 0 {
		ids := ts.pop(int64(max))

		for _, id := range ids {
			t.Log("old", old, "new", id)

			if id >= old {
				old = id
			} else {
				t.Error("timer error")
			}
		}
	}
}
