package rule

import (
	"testing"

	"github.com/Nerdmaster/sendmail/email"
)

func mkrule(t *testing.T, slist ...string) *Rule {
	var r = &Rule{}
	for _, s := range slist {
		var err = r.AddMatcher(s)
		if err != nil {
			t.Fatalf("Error building a rule: matcher %q failed: %s", s, err)
		}
	}
	return r
}

func TestRuleMatch(t *testing.T) {
	var e = email.New()
	e.Header.Set("to", "foo@example.com,Mister F. <tobias.f@example.com>")
	e.Header.Set("from", "somebody@example.com")

	var e2 = email.New()
	e2.Header.Set("from", "me@example.com")

	var regex = mkrule(t, `From/regex:^\w*@example.com$`)
	var failTo = mkrule(t, "To:tobias.f@example.com")
	var successTo = mkrule(t, "To:foo@example.com")
	var catchall = mkrule(t, "*")

	if !catchall.Match(email.New()) || !catchall.Match(e) || !catchall.Match(e2) {
		t.Errorf("catchall should match absolutely any email, even an invalid one")
	}

	if regex.Match(email.New()) {
		t.Errorf("regex shouldn't match an email with no 'from'")
	}

	if !regex.Match(e) {
		t.Errorf("regex (%#v) should have matched the test email (%#v)", regex.matchers[0], e.Header)
	}
	if !regex.Match(e2) {
		t.Errorf("regex (%#v) should have matched the second test email (%#v)", regex, e2)
	}

	if !successTo.Match(e) {
		t.Errorf("success-to (%#v) should have matched test email (%#v)", successTo, e)
	}
	if failTo.Match(e) {
		t.Errorf("fail-to should have matched test email")
	}

	var e3 = email.New()
	e3.Header.Set("from", "somebody+with+non+word+chars@example.com")

	if regex.Match(e3) {
		t.Errorf("regex shouldn't match an email with non-word characters")
	}
}
