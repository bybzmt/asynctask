package scheduler

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	rule_type_direct RuleType = 0
	rule_type_regexp          = 1
)

var rule_types = map[RuleType]string{
	rule_type_direct: "direct",
	rule_type_regexp: "regexp",
}

type RuleType uint

type Rule struct {
	JobConfig

	Pattern string
	Note    string
	Type    RuleType
	Sort    int
	Used    bool

	exp *regexp.Regexp
}

func (c *Rule) init() (err error) {
	c.exp, err = regexp.Compile(c.Pattern)
	return
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

func (s *Scheduler) ruleApply(j *job) {
	j.JobConfig = s.matchRule(j.name)
	j.CmdEnv = copyMap(j.CmdEnv)
	j.HttpHeader = copyMap(j.HttpHeader)

	if j.group != nil {
		jobRemove(j)
	}

	j.group = s.getGroup(j.GroupId)
	j.group.addJob(j)
}

func (s *Scheduler) matchRule(job string) JobConfig {

	var j Rule
	err := db_fetch(s.Db, &j, "config", "rule", rule_types[rule_type_direct], job)
	if err == nil && j.Used {
		return j.JobConfig
	}

	for _, r := range s.rules {
		if r.match(job) {
			return r.JobConfig
		}
	}

	r := Rule{}
	r.Parallel = s.Parallel
	r.GroupId = 1

	return r.JobConfig
}

func (s *Scheduler) rulesReload() error {
	rules, err := s.db_rules_regexp_load()
	if err != nil {
		return err
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Sort > rules[j].Sort
	})

	s.rules = rules

	for _, j := range s.jobs {
		s.ruleApply(j)
	}

	return nil
}

func (s *Scheduler) db_rules_regexp_load() (out []Rule, err error) {

	db_getall(s.Db, func(k, v []byte) error {
		var r Rule
		err := json.Unmarshal(v, &r)
		if err == nil {
			if r.Used {
				if err := r.init(); err == nil {
					out = append(out, r)
				}
			}
		}
		return nil
	}, "config", "rule", rule_types[rule_type_regexp])

	return
}

func (s *Scheduler) RulePut(r Rule) (err error) {
	s.l.Lock()
	defer s.l.Unlock()

	r.Pattern = strings.TrimSpace(r.Pattern)

	if r.Pattern == "" {
		return fmt.Errorf("Pattern empty")
	}

	if r.empty() {
		return fmt.Errorf("task base empty")
	}

	if r.CmdDir != "" {
		r.CmdDir = path.Clean(r.CmdDir)
		if r.CmdDir[0] != '/' {
			return fmt.Errorf("CmdDir Must Absolute Path")
		}
	}

	if _, ok := s.groups[r.GroupId]; !ok {
		return fmt.Errorf("group: %v invalid", r.GroupId)
	}

	switch r.Type {
	case rule_type_direct:
		err = db_put(s.Db, r, "config", "rule", rule_types[rule_type_direct], r.Pattern)
	case rule_type_regexp:
		if _, err = regexp.Compile(r.Pattern); err == nil {
			err = db_put(s.Db, r, "config", "rule", rule_types[rule_type_regexp], r.Pattern)
		}
	default:
		return fmt.Errorf("rule_types: %v invalid", r.Type)
	}

	if err != nil {
		return err
	}

	return s.rulesReload()
}

func (s *Scheduler) RuleDel(t RuleType, pattern string) error {
	s.l.Lock()
	defer s.l.Unlock()

	err := db_del(s.Db, "config", "rule", rule_types[t], pattern)
	if err != nil {
		return err
	}

	return s.rulesReload()
}

func (s *Scheduler) Rules() (out []Rule) {
	s.l.Lock()
	defer s.l.Unlock()

	var r Rule

	db_getall(s.Db, func(k, v []byte) error {
		err := json.Unmarshal(v, &r)
		if err == nil {
			out = append(out, r)
		}
		return nil
	}, "config", "rule", rule_types[rule_type_direct])

	db_getall(s.Db, func(k, v []byte) error {
		err := json.Unmarshal(v, &r)
		if err == nil {
			out = append(out, r)
		}
		return nil
	}, "config", "rule", rule_types[rule_type_regexp])

	return
}
