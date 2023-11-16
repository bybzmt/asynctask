package scheduler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type JobRule struct {
	JobConfig

	Pattern string

	Note string
	Type RuleType
	Sort int
	Used bool

	exp *regexp.Regexp
}

func (c *JobRule) init() (err error) {
	c.exp, err = regexp.Compile(c.Pattern)
	return
}

func (c *JobRule) match(job string) bool {
	if !c.Used {
		return false
	}

	if c.exp == nil {
		return false
	}

	return c.exp.MatchString(job)
}

func (s *Scheduler) ruleApply(j *job) {
	j.JobConfig = s.matchRule(j.name)

	if j.group != nil {
		jobRemove(j)
	}

	j.group = s.getGroup(j.GroupId)
	j.group.addJob(j)
}

func (s *Scheduler) matchRule(job string) JobConfig {

	var j JobRule
	err := db_fetch(s.Db, &j, "config", "jobrule", rule_types[rule_type_direct], job)
	if err == nil && j.Used {
		return j.JobConfig
	}

	for _, r := range s.jobrules {
		if r.match(job) {
			return r.JobConfig
		}
	}

	r := JobRule{}
	r.Parallel = s.Parallel
	r.GroupId = 1

	return r.JobConfig
}

func (s *Scheduler) jobRulesReload() error {
	rules, err := s.db_jobrules_regexp_load()
	if err != nil {
		return err
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Sort > rules[j].Sort
	})

	s.jobrules = rules

	for _, j := range s.jobs {
		s.ruleApply(j)
	}

	return nil
}

func (s *Scheduler) db_jobrules_regexp_load() (out []JobRule, err error) {

	db_getall(s.Db, func(k, v []byte) error {
		var r JobRule
		err := json.Unmarshal(v, &r)
		if err == nil {
			if r.Used {
				if err := r.init(); err == nil {
					out = append(out, r)
				}
			}
		}
		return nil
	}, "config", "jobrule", rule_types[rule_type_regexp])

	return
}

func (s *Scheduler) JobRulePut(r JobRule) (err error) {
	s.l.Lock()
	defer s.l.Unlock()

	r.Pattern = strings.TrimSpace(r.Pattern)

	if r.Pattern == "" {
		return fmt.Errorf("Pattern empty")
	}

	if _, ok := s.groups[r.GroupId]; !ok {
		return fmt.Errorf("group: %v invalid", r.GroupId)
	}

	switch r.Type {
	case rule_type_direct:
		err = db_put(s.Db, r, "config", "jobrule", rule_types[rule_type_direct], r.Pattern)
	case rule_type_regexp:
		if _, err = regexp.Compile(r.Pattern); err == nil {
			err = db_put(s.Db, r, "config", "jobrule", rule_types[rule_type_regexp], r.Pattern)
		}
	default:
		return fmt.Errorf("rule_types: %v invalid", r.Type)
	}

	if err != nil {
		return err
	}

	return s.jobRulesReload()
}

func (s *Scheduler) JobRuleDel(t RuleType, pattern string) error {
	s.l.Lock()
	defer s.l.Unlock()

	err := db_del(s.Db, "config", "jobrule", rule_types[t], pattern)
	if err != nil {
		return err
	}

	return s.jobRulesReload()
}

func (s *Scheduler) JobRules() (out []JobRule) {
	s.l.Lock()
	defer s.l.Unlock()

	db_getall(s.Db, func(k, v []byte) error {
		var r JobRule

		err := json.Unmarshal(v, &r)
		if err == nil {
			out = append(out, r)
		}
		return nil
	}, "config", "jobrule", rule_types[rule_type_direct])

	db_getall(s.Db, func(k, v []byte) error {
		var r JobRule

		err := json.Unmarshal(v, &r)
		if err == nil {
			out = append(out, r)
		}
		return nil
	}, "config", "jobrule", rule_types[rule_type_regexp])

	return
}
