package scheduler

import (
	"encoding/json"
	"regexp"
	"sort"
)

type Rule struct {
	JobConfig

	Id    ID
	Match string
	Sort  int
	Used  bool

	exp *regexp.Regexp
}

func (c *Rule) init() {
	c.exp, _ = regexp.Compile(c.Match)
}

func (c *Rule) match(job string) bool {
	if !c.Used {
		return false
	}

	if c.exp == nil {
		return false
	}

	return c.exp.MatchString(job)
}

func (s *Scheduler) matchRule(job string) *Rule {
	for _, t := range s.rules {
		if t.match(job) {
			return &t
		}
	}

	return nil
}

func (s *Scheduler) db_rule_save(cfg Rule) error {
	val, err := json.Marshal(&cfg)

	if err != nil {
		return err
	}

	//key: config/task/:id
	return db_put(s.Db, val, "config", "task", fmtId(cfg.Id))
}

func (s *Scheduler) rulesChanged() {
	for _, j := range s.jobs {

		if j.cfgMode == 1 {
			continue
		}

		for _, r := range s.rules {
			if r.match(j.Name) {
				if j.GroupId != r.GroupId {
					n := new(job)
					*n = *j
					n.group = s.getGroup(r.GroupId)

					jobRemove(j)
					n.group.addJob(n)
					s.jobs[n.Name] = n
					j = n
				}

				j.JobConfig = r.JobConfig
				j.CmdEnv = copyMap(r.CmdEnv)
				j.HttpHeader = copyMap(r.HttpHeader)
			}
		}
	}
}

func (s *Scheduler) load_rules() error {

	keys, vals := db_getall(s.Db, "config", "task")

	for i, key := range keys {
		val := vals[i]

		var r Rule

		err := json.Unmarshal(val, &r)
		if err != nil {
			s.Log.Warnln("json.Unmarshal:", err, "config", "task", string(key))
		} else {
			r.Id = atoiId(key)
			r.init()

			s.rules = append(s.rules, r)
		}
	}

	s.rulesSort()

	return nil
}

func (s *Scheduler) rulesSort() {
	sort.Slice(s.rules, func(i, j int) bool {
		return s.rules[i].Sort > s.rules[j].Sort
	})
}
