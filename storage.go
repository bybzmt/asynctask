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
	rows := make([]StorageRow, 0, s.jobs.Len())

	s.jobs.Each(func(j *Job) {
		if j.Len() > 0 {
			row := StorageRow{}
			row.Name = j.Name
			row.Contents = make([]string, 0, j.Len())

			ele := j.Tasks.Front()
			for ele != nil {
				row.Contents = append(row.Contents, ele.Value.(*Task).Content)
				ele = ele.Next()
			}

			rows = append(rows, row)
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

	rows := []StorageRow{}

	de := gob.NewDecoder(f)
	err = de.Decode(&rows)
	if err != nil {
		panic(err)
	}

	for _, row := range rows {
		for _, Content := range row.Contents {
			s.AddOrder(row.Name, Content)
		}
	}

	s.e.Log.Println("restore From File complete")
}
