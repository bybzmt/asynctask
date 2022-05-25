package main

import (
	"encoding/gob"
	"os"
)

func (s *Scheduler) saveTask() {
	s.e.Log.Println("[Info] saving tasks...")
	s.saveToFile()
	s.e.Log.Println("[Info] saving tasks complete")
}

func (s *Scheduler) saveToFile() {
	rows := make([]Order, 0, s.WaitNum)

	s.jobs.Each(func(j *Job) {
		if j.Len() > 0 {
			ele := j.Tasks.Front()
			for ele != nil {
				row := Order{}
				row.Name = j.Name
				row.Params = ele.Value.(*Task).Params

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
	s.e.Log.Println("[Info] restore From File")

	f, err := os.Open(*dbfile)
	if err != nil {
		if os.IsNotExist(err) {
			s.e.Log.Println("[Info] not have storaged file")
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
		s.AddOrder(&row)
	}

	s.e.Log.Println("[Info] restore From File complete")
}
