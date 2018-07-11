package email

import (
	"errors"
	"io"
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
			return e.SetupMessage(message)
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
func (e *Email) SetupMessage(message string) error {
	var err error
	e.Message = ""

	for _, line := range lineRegexp.Split(message, -1) {
		// The first blank line means we're done with headers, so there's no more
		// data to be gleaned
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "From: ") {
			if e.From == nil {
				err = e.SetFromAddress(line[6:])
				if err != nil {
					return err
				}
			}

			// Always remove the "From" header to ensure it can be set to e.From just prior to sending the message
			line = ""
		}

		if strings.HasPrefix(line, "Cc: ") {
			err = e.AddToAddresses(line[4:])
			if err != nil {
				return err
			}
		}
		if strings.HasPrefix(line, "Bcc: ") {
			err = e.AddToAddresses(line[5:])
			if err != nil {
				return err
			}
		}
		if strings.HasPrefix(line, "To: ") {
			err = e.AddToAddresses(line[4:])
			if err != nil {
				return err
			}
		}

		e.Message += line
	}

	return nil
}

// SetFromAddress parses addr into a mail.Address, returning an error if
// parsing fails
func (e *Email) SetFromAddress(addr string) error {
	var from, err = mail.ParseAddress(addr)
	if err == nil {
		e.From = from
	}
	return err
}

// AddToAddresses parses addrlist into a list of mail.Addresses, returning an
// error if parsing fails
func (e *Email) AddToAddresses(addrlist string) error {
	var addrs, err = mail.ParseAddressList(addrlist)
	if err == nil {
		e.To = append(e.To, addrs...)
	}
	return err
}

// Send uses the from/to/message data combined with host to attempt to send the
// message via smtp
func (e *Email) Send(host string) error {
	var toList = make([]string, len(e.To))
	for i, to := range e.To {
		toList[i] = to.String()
	}

	if e.From == nil || len(e.To) == 0 {
		return errors.New("mail.Send: must have from and to addresses set")
	}

	// Hack in the "From" header
	var msg = "From: " + e.From.String() + "\r\n" + e.Message

	var err = smtp.SendMail(host, e.Auth, e.From.String(), toList, []byte(msg))
	return err
}
