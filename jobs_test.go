package main

import (
	"fmt"
	"testing"
)

func TestList(t *testing.T) {
	s := &Scheduler{}
	js := &Jobs{}
	js.Init(10, s)

	a := &Job{Name: "a1"}
	js.PushBack(a)
	fmt.Println("size", js.Len())

	b := &Job{Name: "a2"}
	js.PushBack(b)
	fmt.Println("size", js.Len())

	c := &Job{Name: "a3"}
	js.PushBack(c)
	fmt.Println("size", js.Len())

	fmt.Println(js)
	js.MoveBefore(c, a)

	fmt.Println(js)

	js.Remove(b)
	js.Remove(a)
	js.Remove(c)

	if js.Len() != 0 {
		t.Fatal("jobs list size err")
	}
}
