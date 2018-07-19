package rule

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nerdmaster/sendmail/email"
)

type matcher struct {
	field    string
	value    string
	catchall bool
	regex    bool
	re       *regexp.Regexp
}

func newMatcher(condition string) (*matcher, error) {
	if condition == "*" {
		return &matcher{catchall: true}, nil
	}

	var parts = strings.SplitN(condition, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("sendmail/filter: match condition format must have a colon")
	}

	var m = &matcher{field: strings.ToLower(parts[0]), value: parts[1]}
	if strings.HasSuffix(m.field, "/regex") {
		m.field = m.field[:len(m.field)-6]
		m.regex = true
		var err error
		m.re, err = regexp.Compile(m.value)
		if err != nil {
			return nil, fmt.Errorf("sendmail/filter: invalid match condition regex: %s", err)
		}
	}

	return m, nil
}

func (m *matcher) match(e *email.Email) bool {
	if m.catchall {
		return true
	}

	var val string
	switch m.field {
	// For email-containing fields, we attempt to grab an address and use that.
	// We don't care about errors here, because lack of a valid address just
	// means it can't match *anything*.
	case "to", "from", "cc", "bcc":
		var addr, err = e.Header().Address(m.field)
		if err != nil || addr == nil {
			return false
		}
		val = addr.Address
	default:
		val = e.Header().Get(m.field)
	}

	if m.regex {
		return m.re.MatchString(val)
	}
	return m.value == val
}

// A Rule is a collection of match directives to determine if an email should
// be handled
type Rule struct {
	matchers []*matcher
}

func (r *Rule) AddMatcher(condition string) error {
	var m, err = newMatcher(condition)
	if err != nil {
		return err
	}
	r.matchers = append(r.matchers, m)
	return nil
}

// Match returns true if all matchers match the given email
func (r *Rule) Match(e *email.Email) bool {
	if len(r.matchers) == 0 {
		return false
	}

	for _, matcher := range r.matchers {
		if !matcher.match(e) {
			return false
		}
	}
	return true
}
