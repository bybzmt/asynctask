package scheduler

import (
	"encoding/json"
	"fmt"
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

type TaskRule struct {
	TaskBase

	Pattern     string
	RewriteReg  string
	RewriteRepl string

	Note string
	Type RuleType
	Sort int
	Used bool

	exp  *regexp.Regexp
	exp2 *regexp.Regexp
}

func (c *TaskRule) init() (err error) {
	c.exp, err = regexp.Compile(c.Pattern)
	if err != nil {
		return
	}

	if c.RewriteReg != "" {
		c.exp2, err = regexp.Compile(c.RewriteReg)
	}

	return
}

func (c *TaskRule) match(job string) bool {
	if !c.Used {
		return false
	}

	if c.exp == nil {
		return false
	}

	return c.exp.MatchString(job)
}

func (s *Scheduler) matchTask(name string) (string, *TaskRule) {

	var j TaskRule
	err := db_fetch(s.Db, &j, "config", "taskrule", rule_types[rule_type_direct], name)
	if err == nil && j.Used {
		return name, &j
	}

	for _, r := range s.taskrules {
		p := r.exp.FindStringSubmatch(name)

		l := len(p)

		if l == 1 {
			return name, &r
		} else if l > 1 {
			return p[1], &r
		}
	}

	return "", nil
}

func (s *Scheduler) taskRulesReload() error {
	rules, err := s.db_taskrules_regexp_load()
	if err != nil {
		return err
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Sort > rules[j].Sort
	})

	s.taskrules = rules

	return nil
}

func (s *Scheduler) db_taskrules_regexp_load() (out []TaskRule, err error) {

	db_getall(s.Db, func(k, v []byte) error {
		var r TaskRule
		err := json.Unmarshal(v, &r)
		if err == nil {
			if r.Used {
				if err := r.init(); err == nil {
					out = append(out, r)
				}
			}
		}
		return nil
	}, "config", "taskrule", rule_types[rule_type_regexp])

	return
}

func (s *Scheduler) TaskRulePut(r TaskRule) (err error) {
	s.tl.Lock()
	defer s.tl.Unlock()

	r.Pattern = strings.TrimSpace(r.Pattern)

	if r.Pattern == "" {
		return fmt.Errorf("Pattern empty")
	}

	switch r.Type {
	case rule_type_direct:
		err = db_put(s.Db, r, "config", "taskrule", rule_types[rule_type_direct], r.Pattern)
	case rule_type_regexp:
		if _, err = regexp.Compile(r.Pattern); err == nil {
			err = db_put(s.Db, r, "config", "taskrule", rule_types[rule_type_regexp], r.Pattern)
		}
	default:
		return fmt.Errorf("rule_types: %v invalid", r.Type)
	}

	if err != nil {
		return err
	}

	return s.taskRulesReload()
}

func (s *Scheduler) TaskRuleDel(t RuleType, pattern string) error {
	s.tl.Lock()
	defer s.tl.Unlock()

	err := db_del(s.Db, "config", "taskrule", rule_types[t], pattern)
	if err != nil {
		return err
	}

	return s.taskRulesReload()
}

func (s *Scheduler) TaskRules() (out []TaskRule) {
	s.tl.Lock()
	defer s.tl.Unlock()

	var r TaskRule

	db_getall(s.Db, func(k, v []byte) error {
		err := json.Unmarshal(v, &r)
		if err == nil {
			out = append(out, r)
		}
		return nil
	}, "config", "taskrule", rule_types[rule_type_direct])

	db_getall(s.Db, func(k, v []byte) error {
		err := json.Unmarshal(v, &r)
		if err == nil {
			out = append(out, r)
		}
		return nil
	}, "config", "taskrule", rule_types[rule_type_regexp])

	return
}
