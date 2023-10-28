package scheduler

import (
	"crypto/rand"
	"log"
	"sort"
	"testing"
)

func TestPriority(t *testing.T) {

	var scores []int
	g := new(group)
    g.s = new(Scheduler)
	g.init()

	for i := 0; i < 20; i++ {
		scores = append(scores, ts_getRand()%100)
	}

	for _, score := range scores {
		j := new(job)
		j.score = score

		g.runAdd(j)
		g.priority(j)

		log.Printf("tmp %v", ts_getJobsScore(g))
	}

	sort.Ints(scores)

	jscores := ts_getJobsScore(g)

	if len(scores) != len(jscores) {
		t.Error("scores len error")
		return
	}

	for i, score := range scores {
		if jscores[i] != score {
			t.Error("scores len error")

			log.Printf("expect %v", scores)
			log.Printf("got %v", jscores)
			return
		}
	}
}

func ts_getRand() int {
	b := make([]byte, 4)
	rand.Read(b)
	num := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
	return num
}

func ts_getJobsScore(g *group) []int {
	var jscores []int

	for j := g.run.next; j != g.run; j = j.next {
		jscores = append(jscores, j.score)
	}

    return jscores
}
