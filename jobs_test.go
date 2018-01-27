package main

import (
	"fmt"
	"testing"
	"log"
	"os"
	"time"
)

func TestList(t *testing.T) {
	std := log.New(os.Stdout, "[Info] ", log.LstdFlags)
	err := log.New(os.Stderr, "[Scheduler] ", log.LstdFlags)

	hub = new(Scheduler).Init(10, "", std, err)

	js := &hub.jobs

	a := &Order{Method:"GET", Name: "a1"}
	js.AddTask(a)

	b := &Order{Method:"GET", Name: "a2"}
	js.AddTask(b)

	c := &Order{Method:"GET", Name: "a3"}
	js.AddTask(c)

	now := time.Now()

	js.GetTask(now);
	js.GetTask(now);
	js.GetTask(now);

	fmt.Println(js)

	if js.HasTask() {
		t.Fatal("jobs list err")
	}

}
