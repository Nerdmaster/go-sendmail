package main

import (
	"testing"

	"github.com/Nerdmaster/sendmail/email"
)

func TestAuthMatch(t *testing.T) {
	var e = &email.Email{}
	e.SetToAddresses("foo@example.com,Mister F. <tobias.f@example.com>")
	e.SetFromAddress("somebody@example.com")

	var regexAuth = &authorization{FromRegex: `^\w*@example.com$`}
	var failToAuth = &authorization{To: "tobias.f@example.com"}
	var successToAuth = &authorization{To: "foo@example.com"}
	var catchallAuth = &authorization{Catchall: true}

	if !catchallAuth.matches(email.New()) {
		t.Errorf("catchall should match absolutely any email, even an invalid one")
	}

	if regexAuth.matches(email.New()) {
		t.Errorf("regex shouldn't match an email with no 'from'")
	}

	if !regexAuth.matches(e) {
		t.Errorf("regex should have matched the test email")
	}

	if !successToAuth.matches(e) {
		t.Errorf("success-to should have matched test email")
	}
	if failToAuth.matches(e) {
		t.Errorf("fail-to should have matched test email")
	}

	var e2 = email.New()
	e2.SetFromAddress("somebody+with+non+word+chars@example.com")

	if regexAuth.matches(e2) {
		t.Errorf("regex shouldn't match an email with non-word characters")
	}
}

func TestAuthSet(t *testing.T) {
	var e = email.New()
	var rewriterAuth = &authorization{Catchall: true, RewriteFrom: `"Flap-E the Bun-E" <blah@example.com>`}

	rewriterAuth.assignToEmail(e)
	if e.Auth == nil {
		t.Errorf("e.Auth should have been assigned")
	}
	if e.From == nil {
		t.Errorf("e.From should have been set")
	}
	if e.From.String() != rewriterAuth.RewriteFrom {
		t.Errorf("e.From should have been %q but was %q", rewriterAuth.RewriteFrom, e.From.String())
	}
}
