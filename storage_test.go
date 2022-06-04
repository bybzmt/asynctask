package main

import (
	"log"
	"os"
	"testing"
)

func TestStorage(t *testing.T) {
	f, err := os.CreateTemp("", "tmp")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name()) // clean up

	var cfg Config
	var hub Scheduler

	cfg.DbFile = f.Name()
	hub.Init(&cfg)

	for i := uint(0); i < 10; i++ {
		o := Order{
			Id:   i,
			Name: "test",
		}
		hub.addTask(&o)
	}

	hub.saveToFile()

	var hub2 Scheduler
	hub2.Init(&cfg)
	hub2.running = true

	go hub2.restoreFromFile()

	for i := uint(0); i < 10; i++ {
		o := <-hub2.order

		t.Log("order", o)

		if i != o.Id {
			t.Fatal("Storage restore error", i)
		}
	}
}
