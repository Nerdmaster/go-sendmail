package email

import (
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
