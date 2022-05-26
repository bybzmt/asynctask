package main

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestList(t *testing.T) {
	logger := log.Default()

	env := new(Environment).Init(10, "", 10, logger)
	hub := new(Scheduler).Init(env)

	js := &hub.jobs

	a := &Order{Name: "a1"}
	js.AddTask(a)

	b := &Order{Name: "a2"}
	js.AddTask(b)

	c := &Order{Name: "a3"}
	js.AddTask(c)

	now := time.Now()

	js.GetTask(now)
	js.GetTask(now)
	js.GetTask(now)

	fmt.Println(js)

	if js.HasTask() {
		t.Fatal("jobs list err")
	}

}
