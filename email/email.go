package email

import (
	"errors"
	"io"
	"log"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
)

var lineRegexp = regexp.MustCompile("\r\n|\r|\n")

// An Email parses message data to prepare for SMTP delivery
type Email struct {
	From    *mail.Address
	To      []*mail.Address
	Message string
	Auth    smtp.Auth
}

// New returns a new empty Email pointer
func New() *Email {
	return new(Email)
}

// Read processes the given reader, treating it as if it were a stdin buffer as
// sendmail does.  It will automatically split up lines on any kind of newline,
// and then process headers via SetupMessage.
func (e *Email) Read(r io.Reader) error {
	var eof bool
	var rawMessage []byte
	var message string
	var buf [10240]byte
	for !eof {
		var xbuf = buf[0:]
		var n, err = r.Read(xbuf)
		if err != nil && err != io.EOF {
			return err
		}

		rawMessage = append(rawMessage, xbuf[:n]...)
		message, eof = bytesToMessage(rawMessage)
		if eof || err == io.EOF {
			e.SetupMessage(message)
			return nil
		}
	}

	return errors.New("email.Read: no data")
}

func bytesToMessage(rawMessage []byte) (message string, done bool) {
	var s = string(rawMessage)
	var lines = lineRegexp.Split(s, -1)

	// Kill the trailing newline
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for i, line := range lines {
		if line == "." {
			return strings.Join(lines[:i], "\r\n"), true
		}
	}
	return strings.Join(lines, "\r\n"), false
}

// SetupMessage stores the message and then looks for headers in order to
// determine from/to in case those weren't passed on the command line
func (e *Email) SetupMessage(message string) {
	e.Message = message
	for _, line := range lineRegexp.Split(message, -1) {
		// The first blank line means we're done with headers, so there's no more data to be gleaned
		if line == "" {
			return
		}

		if e.From == nil && strings.HasPrefix(line, "From: ") {
			e.SetFromAddress(line[6:])
		}

		if strings.HasPrefix(line, "Cc: ") {
			e.AddToAddresses(line[4:])
		}
		if strings.HasPrefix(line, "Bcc: ") {
			e.AddToAddresses(line[5:])
		}
		if strings.HasPrefix(line, "To: ") {
			e.AddToAddresses(line[4:])
		}
	}
}

// SetFromAddress parses addr into a mail.Address, returning an error if parsing fails
func (e *Email) SetFromAddress(addr string) {
	var err error
	e.From, err = mail.ParseAddress(addr)
	if err != nil {
		log.Fatalf(`Invalid "from" address %q: %s`, addr, err)
	}
}

func (e *Email) AddToAddresses(addrlist string) {
	var addrs, err = mail.ParseAddressList(addrlist)
	if err != nil {
		log.Fatalf("Invalid address(es) %q: %s", addrlist, err)
	}
	e.To = append(e.To, addrs...)
}
