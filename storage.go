package main

import (
	"encoding/gob"
	"flag"
	"os"
)

var dbfile = flag.String("dbfile", "./asynctask.db", "storage file")

type StorageRow struct {
	Name     string
	Contents []string
}

func (s *Scheduler) saveToFile() {

	rows := []StorageRow{}
	rows.Contents = make([]string, 0, s.WaitNum)

	for _, ele := range s.jobs.all.all {
		j, ok := ele.Value.(*lruKv).val.(*Job)
		if ok {
			rows.Name = j.Name

			ele := j.Tasks.Front()
			for ele != nil {
				rows.Contents = append(rows.Contents, ele.Value.(Task).Content)
				ele = ele.Next()
			}
		}
	}

	f, err := os.OpenFile(*dbfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	en := gob.NewEncoder(f)
	err = en.Encode(rows)
	if err != nil {
		panic(err)
	}
}

func (s *Scheduler) restoreFromFile() {
	s.e.Log.Println("restore From File")

	f, err := os.Open(*dbfile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		panic(err)
	}
	defer f.Close()

	rows := StorageRow{}
	rows.Contents = make([]string, 0, s.WaitNum)

	de := gob.NewDecoder(f)
	err = de.Decode(&rows)
	if err != nil {
		panic(err)
	}

	for _, o := range orders {
		s.AddOrder(o.Name, o.Content)
	}

	s.e.Log.Println("restore From File complete")
}
