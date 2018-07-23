package email

import (
	"bytes"
	"net/mail"
	"net/smtp"
	"strings"
	"testing"

	"github.com/uoregon-libraries/gopkg/assert"
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

func getaddrs(e *Email, t *testing.T) (from *mail.Address, tolist AddressList, cclist AddressList) {
	var err error

	from, err = e.Header.Address("from")
	if err != nil || from.Address == "" {
		t.Logf("%#v", e)
		t.Fatalf("Error: from header is empty")
	}
	tolist, err = e.Header.AddressList("to")
	if err != nil || len(tolist) == 0 {
		t.Logf("%#v", e)
		t.Fatalf("Error: to header is empty")
	}
	cclist, err = e.Header.AddressList("cc")
	if err != nil || len(cclist) == 0 {
		t.Logf("%#v", e)
		t.Fatalf("Error: cc header is empty")
	}

	return
}

func TestRead(t *testing.T) {
	var e, err = Read(bytes.NewBufferString("Subject: hello\nTo: you <you@example.org>\n" +
		"From: me <me@example.org>\nCc: her <her@example.org>,bobby@tables.example.org\n\nhi"))
	if err != nil {
		t.Fatalf("Couldn't read email: %s", err)
	}

	var from, tolist, cclist = getaddrs(e, t)

	assert.Equal(`"me" <me@example.org>`, from.String(), "from address", t)
	assert.Equal(1, len(tolist), "to address count", t)
	assert.Equal("you", tolist[0].Name, "to name", t)
	assert.Equal("you@example.org", tolist[0].Address, "to address", t)
	assert.Equal(2, len(cclist), "cc address count", t)
	assert.Equal("her", cclist[0].Name, "1st cc name", t)
	assert.Equal("", cclist[1].Name, "2nd cc name", t)
	assert.Equal("her@example.org", cclist[0].Address, "1st cc address", t)
	assert.Equal("bobby@tables.example.org", cclist[1].Address, "2nd cc address", t)
	assert.Equal("hi", string(e.Message), "message", t)
}

func TestIgnoresDupeFields(t *testing.T) {
	var e, err = Read(bytes.NewBufferString("Subject: hello\nTo: you@example.org\n" +
		"From: me@example.org\nFrom: her <her@example.org>\n\nHello there!"))
	if err != nil {
		t.Fatalf("Couldn't read email: %s", err)
	}
	var buf = new(bytes.Buffer)
	e.Header.Write(buf)
	assert.Equal("From: me@example.org\r\nSubject: hello\r\nTo: you@example.org",
		string(buf.Bytes()), "Header shows only the first From field", t)
}

func TestSend(t *testing.T) {
	var f = new(fakeSentMessage)
	var e = New()
	e.Mailer = f.fakeMail
	e.read(bytes.NewBufferString("To: Another cow <another+cow@example.org>\n" +
		"CC: one@example.org,two@example.org\n" +
		"bcc: uno@example.org\n" +
		"Subject: Blah\n\n" +
		"Hello!"))
	e.Header.Set("from", "Chicken <chicken@example.org>")

	var host = "host:25"
	e.Send(host)
	assert.Equal(host, f.addr, "host", t)
	assert.Equal(`"Chicken" <chicken@example.org>`, f.from, "from", t)
	assert.Equal(`"Another cow" <another+cow@example.org>,<one@example.org>,<two@example.org>,<uno@example.org>`, strings.Join(f.to, ","), "to", t)
	assert.Equal(
		"Cc: one@example.org,two@example.org\r\n"+
			"From: Chicken <chicken@example.org>\r\n"+
			"Subject: Blah\r\n"+
			"To: Another cow <another+cow@example.org>\r\n"+
			"\r\nHello!", string(f.msg), "massaged message: 'from' header added, 'bcc' removed, sorted headers", t)
}

func TestHeaders(t *testing.T) {
	var e = New()
	e.Header.Set("from", "user@example.org")
	assert.Equal("user@example.org", e.Header.Get("from"), "from header is set properly", t)
	e.Header.Del("from")
	assert.Equal("", e.Header.Get("from"), "from header is removed properly", t)
}
