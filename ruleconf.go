package main

import (
	"log"

	"github.com/Nerdmaster/sendmail/rule"
)

type authentication struct {
	Host     string
	Username string
	Password string
	Server   string
}

// RuleConf is a config-friendly composition for making config strings turn into
// rule.Rules and living alongside the smtp auth we need for sending emails
type RuleConf struct {
	rule     *rule.Rule
	Matchers []string
	Actions  []string
	Auth     *authentication
}

func (r *RuleConf) initRule() {
	r.rule = new(rule.Rule)
	for _, mstr := range r.Matchers {
		var err = r.rule.AddMatcher(mstr)
		if err != nil {
			log.Fatalf("Invalid rule matcher string (%s): %s", mstr, err)
		}
	}
	for _, astr := range r.Actions {
		var err = r.rule.AddAction(astr)
		if err != nil {
			log.Fatalf("Invalid rule action string (%s): %s", astr, err)
		}
	}
}

// initRules takes the configuration parts of the RuleConf and creates the
// concrete rule.Rule definitions
func initRules(rlist []*RuleConf) {
	for _, r := range rlist {
		r.initRule()
	}
}
