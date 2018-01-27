package main

import (
	"encoding/gob"
	"flag"
	"os"
)

var dbfile = flag.String("dbfile", "./asynctask.db", "storage file")

func (s *Scheduler) saveTask() {
	s.e.Log.Println("saving tasks...")
	s.saveToFile()
	s.e.Log.Println("saving tasks complete")
}

func (s *Scheduler) saveToFile() {
	rows := make([]Order, 0, s.WaitNum)

	s.jobs.Each(func(j *Job) {
		if j.Len() > 0 {
			ele := j.Tasks.Front()
			for ele != nil {
				row := Order{}
				row.Method = j.Method
				row.Name = j.Name
				row.Content = ele.Value.(*Task).Content

				rows = append(rows, row)

				ele = ele.Next()
			}
		}
	})

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
			s.e.Log.Println("not have storaged file")
			return
		}
		panic(err)
	}
	defer f.Close()

	rows := []Order{}

	de := gob.NewDecoder(f)
	err = de.Decode(&rows)
	if err != nil {
		panic(err)
	}

	for _, row := range rows {
		s.AddOrder(row.Method, row.Name, row.Content)
	}

	s.e.Log.Println("restore From File complete")
}
