package main

import (
	"testing"
)

func TestList(t *testing.T) {
	env := new(Config)
	env.Init("http", 10)
	hub := new(Scheduler).Init(env)

	js := &hub.jobs

	a := &Order{Name: "a1"}
	js.AddTask(a)

	b := &Order{Name: "a2"}
	js.AddTask(b)

	c := &Order{Name: "a3"}
	js.AddTask(c)

	js.GetTask()
	js.GetTask()
	js.GetTask()

	//fmt.Println(js)

	if js.HasTask() {
		t.Fatal("jobs list err")
	}

}
