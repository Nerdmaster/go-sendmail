package rule

import (
	"errors"

	"github.com/Nerdmaster/sendmail/email"
)

type action interface {
	apply(e *email.Email)
}

// AddAction parses the string and puts the parsed action into the Rule's
// actions list.  If parsing fails, an error is returned.
func (r *Rule) AddAction(astr string) error {
	return errors.New("not implemented")
}

type actSetHeader struct {
	field  string
	newval string
}

func (a *actSetHeader) apply(e *email.Email) {
	e.Header.Set(a.field, a.newval)
}
