package email

import (
	"net/smtp"
	"strings"
	"testing"

	"github.com/Nerdmaster/sendmail/assert"
)

type fakeSentMessage struct {
	addr string
	a    smtp.Auth
	from string
	to   []string
	msg  []byte
}

func (f *fakeSentMessage) fakeMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	f.addr = addr
	f.a = a
	f.from = from
	f.to = to
	f.msg = msg

	return nil
}

func TestSend(t *testing.T) {
	var e = New()
	var f = new(fakeSentMessage)
	e.Mailer = f.fakeMail
	e.SetFromAddress("Chicken <chicken@example.com>")
	e.AddToAddresses("foo@example.com,bar@example.com")
	e.AddToAddresses("Cow <cow@example.com>")
	e.SetupMessage("To: Another cow <another+cow@example.com>\nSubject: Blah\n\nHello!")

	var host = "host:25"
	e.Send(host)
	assert.Equal(host, f.addr, "host", t)
	assert.Equal(`"Chicken" <chicken@example.com>`, f.from, "from", t)
	assert.Equal(`<foo@example.com>,<bar@example.com>,"Cow" <cow@example.com>,"Another cow" <another+cow@example.com>`, strings.Join(f.to, ","), "to", t)
	assert.Equal("From: \"Chicken\" <chicken@example.com>\r\n" +
		"To: Another cow <another+cow@example.com>\r\n" +
		"Subject: Blah\r\n\r\nHello!", string(f.msg), "massaged message with 'from' header added", t)
}
