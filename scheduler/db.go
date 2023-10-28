package scheduler

import (
	"encoding/json"
)

func (s *Scheduler) db_Job_load(job string) (out *JobConfig) {
	//key: config/job/:name
	v := db_get(s.Db, "config", "job", job)
	if v == nil {
		return nil
	}

	err := json.Unmarshal(v, &out)
	if err != nil {
		out = nil
		s.Log.Warnln("loadConfig", job, "Unmarshal error")
	}

	return
}

func (s *Scheduler) db_Job_save(c *JobConfig) error {
	v, err := json.Marshal(c)
	if err != nil {
		return err
	}

	//key: config/job/:name
	return db_put(s.Db, v, "config", "job", c.Name)
}

func (s *Scheduler) db_Job_del(c *JobConfig) error {
	//key: config/job/:name
	return db_del(s.Db, "config", "job", c.Name)
}

func (s *Scheduler) db_group_save(g *group) error {
	val, err := json.Marshal(&g.GroupConfig)
	if err != nil {
		return err
	}

	//key: config/group/:id
	return db_put(s.Db, val, "config", "group", fmtId(g.Id))
}
