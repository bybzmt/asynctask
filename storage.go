package main

import (
	"encoding/gob"
	"os"
)

func (s *Scheduler) saveTask() {
	s.log.Println("[Info] saving tasks...")
	if s.redis != nil {
		s.saveToRedis()
	} else {
		s.saveToFile()
	}
	s.log.Println("[Info] saving tasks complete")
}

func (s *Scheduler) saveToFile() {
	rows := make([]Order, 0, s.WaitNum)

	s.jobs.Each(func(j *Job) {
		if j.Len() > 0 {
			ele := j.Tasks.Front()
			for ele != nil {
				t := ele.Value.(*Task)

				row := Order{
					Id:       t.Id,
					Parallel: j.parallel,
					Name:     j.Name,
					Params:   t.Params,
					AddTime:  uint(t.AddTime.Unix()),
				}

				rows = append(rows, row)

				ele = ele.Next()
			}
		}
	})

	f, err := os.OpenFile(s.cfg.DbFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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
	s.log.Println("[Info] restore From File")

	f, err := os.Open(s.cfg.DbFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.log.Println("[Info] not have storaged file")
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

	s.log.Println("[Info] restore From File complete")
}
