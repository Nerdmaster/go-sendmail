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
	Auth     *authentication
}

// initRules takes the configuration parts of the RuleConf and creates the
// concrete rule.Rule definitions
func initRules(rlist []*RuleConf) {
	var err error
	for i, r := range rlist {
		r.rule = new(rule.Rule)
		for _, mstr := range r.Matchers {
			err = r.rule.AddMatcher(mstr)
			if err != nil {
				log.Fatalf("Invalid rule configuration (rule %d; matcher %q): %s", i, mstr, err)
			}
		}
	}
}