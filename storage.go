package main

import (
	"encoding/json"
	"io"
	"os"
)

func (s *Scheduler) saveTask() {
	s.log.Println("[Info] saving tasks...")
	if s.redis != nil {
		s.saveToRedis()
		s.redis.Close()
	} else {
		s.saveToFile()
	}
	s.log.Println("[Info] saving tasks complete")
}

func (s *Scheduler) saveToFile() {
	f, err := os.OpenFile(s.cfg.DbFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)

	s.jobs.Each(func(j *Job) {
		j.Each(func(t *taskMini) {
			row := Order{
				Id:       t.Id,
				Parallel: j.parallel,
				Name:     j.Name,
				Params:   t.Params,
				AddTime:  t.AddTime,
			}
			err := encoder.Encode(&row)
			if err != nil {
				s.log.Panicln("[Error] json encode error", err)
			}
		})
	})
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

	decoder := json.NewDecoder(f)

	for {
		row := Order{}

		err := decoder.Decode(&row)
		if err != nil {
			if err == io.EOF {
				s.log.Println("[Info] restore From File complete")
				return
			}

			s.log.Println("[Error] json decode error", err)
			return
		} else {
			s.AddOrder(&row)
		}
	}
}
