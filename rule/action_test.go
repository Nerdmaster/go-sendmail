package rule

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Nerdmaster/sendmail/email"
	"github.com/uoregon-libraries/gopkg/assert"
)

func mkActRule(t *testing.T, alist ...string) *Rule {
	var r = &Rule{}
	for _, s := range alist {
		var err = r.AddAction(s)
		if err != nil {
			t.Fatalf("Error building a rule: action %q failed: %s", s, err)
		}
	}
	return r
}

func TestRuleActionSetHeader(t *testing.T) {
	var e = email.New()
	e.Header.Set("to", "foo@example.com,Mister F. <tobias.f@example.com>")
	e.Header.Set("from", "somebody@example.com")

	var r = mkActRule(t, "SetHeader cc:me@example.com", `SetHeader reply-to:--{{.Get "FROM"}}--`)
	r.Apply(e)
	var b bytes.Buffer
	e.Header.Write(&b)
	var lines = strings.Split(b.String(), "\r\n")
	assert.Equal("Cc: me@example.com", lines[0], "header line 0", t)
	assert.Equal("From: somebody@example.com", lines[1], "header line 1", t)
	assert.Equal("Reply-To: --somebody@example.com--", lines[2], "header line 2", t)
	assert.Equal("To: foo@example.com,Mister F. <tobias.f@example.com>", lines[3], "header line 3", t)
}
