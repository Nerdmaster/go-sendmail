package rule

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"github.com/Nerdmaster/sendmail/email"
)

type action interface {
	apply(e *email.Email)
}

// AddAction parses the string and puts the parsed action into the Rule's
// actions list.  If parsing fails, an error is returned.
func (r *Rule) AddAction(astr string) error {
	var parts = strings.SplitN(astr, " ", 2)
	if len(parts) != 2 {
		return errors.New("invalid action syntax")
	}
	switch parts[0] {
	case "SetHeader":
		var a, err = newActSetHeader(parts[1])
		if err == nil {
			r.actions = append(r.actions, a)
		}
		return err
	}
	return errors.New("unknown action command: " + parts[0])
}

type actSetHeader struct {
	field string
	tmpl  *template.Template
}

func newActSetHeader(data string) (*actSetHeader, error) {
	var parts = strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid SetHeader syntax: missing new value")
	}
	var a = &actSetHeader{field: parts[0]}
	a.tmpl = template.New("tmpl")
	var _, err = a.tmpl.Parse(parts[1])
	if err != nil {
		return nil, errors.New("invalid SetHeader syntax: " + err.Error())
	}

	return a, nil
}

func (a *actSetHeader) apply(e *email.Email) {
	var b bytes.Buffer
	a.tmpl.Execute(&b, e.Header)
	e.Header.Set(a.field, b.String())
}
