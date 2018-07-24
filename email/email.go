package email

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
)

var lineRegexp = regexp.MustCompile("\r\n|\r|\n")

// Header is a read-only mail.Header wrapper
type Header struct {
	h mail.Header
}

// Address returns a single address for the given header field.  Uses h.Get, so
// if there is more than one value, only the first is used.  If there is more
// than one email address in the list, only the first is returned.  Suitable
// for pulling the "From" field, which is typically only one address, or the
// first (and therefore hopefully the most important) "To" field.
func (h Header) Address(key string) (addr *mail.Address, err error) {
	var list AddressList
	list, err = h.AddressList(key)
	if len(list) > 0 {
		addr = list[0]
	}

	return addr, err
}

// AddressList parses the named header field as a list of addresses.
func (h Header) AddressList(key string) (AddressList, error) {
	var list, err = h.h.AddressList(key)

	// Ignore errors when it's simply a lack of a given header
	if err == mail.ErrHeaderNotPresent {
		err = nil
	}

	return list, err
}

// Get returns the value for the given header field
func (h Header) Get(key string) string {
	return textproto.MIMEHeader(h.h).Get(key)
}

// Set replaces the field identified by key with the single value passed in
func (h Header) Set(key, value string) {
	textproto.MIMEHeader(h.h).Set(key, value)
}

// Del removes the given header
func (h Header) Del(key string) {
	textproto.MIMEHeader(h.h).Del(key)
}

// Write writes a header in wire format, reprinting only the first value of any
// given field, as the spec states multiple occurrences of the same field have
// no specific interpretation, and are discouraged.  We deliberately ignore BCC
// since outgoing emails don't need it.
func (h Header) Write(w io.Writer) error {
	var data []string
	for k, vlist := range h.h {
		if strings.ToLower(k) == "bcc" {
			continue
		}
		data = append(data, k+": "+vlist[0])
	}
	sort.Strings(data)
	var _, err = w.Write([]byte(strings.Join(data, "\r\n")))
	return err
}

// Clone creates a copy of all keys and their value lists
func (h Header) Clone() Header {
	var h2 = Header{make(mail.Header)}
	for k, vlist := range h.h {
		h2.h[k] = make([]string, len(vlist))
		copy(h2.h[k], h.h[k])
	}

	return h2
}

// AddressList aliases a slice of addresses to help with "round-tripping" a
// list back into strings
type AddressList []*mail.Address

// String returns a parseable string of email addresses
func (list AddressList) String() string {
	return strings.Join(list.Strings(), ",")
}

// Strings returns a single string for each address in the list, suitable for
// the smtp SendMail call
func (list AddressList) Strings() []string {
	var s = make([]string, len(list))
	for i, addr := range list {
		s[i] = addr.String()
	}
	return s
}

// An Email parses message data to prepare for SMTP delivery
type Email struct {
	Message []byte
	Header  Header
	Auth    smtp.Auth
	Mailer  func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// New returns a basic Email instance with its Mailer set to the default smtp.SendMail
func New() *Email {
	return &Email{Mailer: smtp.SendMail, Header: Header{h: make(mail.Header)}}
}

// Read processes the given reader, treating it as if it were a stdin buffer as
// sendmail does.  Headers which set From, To, CC, or BCC values will set those
// fields in the returned Email instance.
func Read(r io.Reader) (*Email, error) {
	var e = New()
	return e, e.read(r)
}

// read actually does the work of parsing data from r
func (e *Email) read(r io.Reader) error {
	// In order to stop on the first ".", we have to process and then rewrite the
	// reader, otherwise the mail package just keeps on reading indefinitely.
	// Not to mention includes that "." in the email body.
	var newR, err = e.readToDot(r)
	if err != nil {
		return err
	}

	var m *mail.Message
	m, err = mail.ReadMessage(newR)
	if err != nil {
		return err
	}

	e.Header.h = m.Header
	e.Message, err = ioutil.ReadAll(m.Body)
	if err != nil {
		return err
	}

	return nil
}

// readToDot reads the raw email data until a single "." is on a line by itself
// or the stream ends
func (e *Email) readToDot(r io.Reader) (io.Reader, error) {
	var s = bufio.NewScanner(r)
	var lines []string
	for s.Scan() {
		var txt = s.Text()
		if txt == "." {
			break
		}
		lines = append(lines, s.Text())
	}

	if s.Err() != nil {
		return nil, errors.New(`mail: scanner reported error reading email: ` + s.Err().Error())
	}

	return strings.NewReader(strings.Join(lines, "\r\n")), s.Err()
}

// Send uses the header data, Auth, and the given host to attempt to send the
// message via smtp
func (e *Email) Send(host string) error {
	var err error
	var from *mail.Address
	var to, cc, bcc AddressList

	from, err = e.Header.Address("from")
	if err != nil {
		return errors.New(`mail.Send: invalid "from" field: ` + err.Error())
	}
	to, err = e.Header.AddressList("to")
	if err != nil {
		return errors.New(`mail.Send: invalid "to" field: ` + err.Error())
	}
	cc, err = e.Header.AddressList("cc")
	if err != nil {
		return errors.New(`mail.Send: invalid "cc" field: ` + err.Error())
	}
	bcc, err = e.Header.AddressList("bcc")
	if err != nil {
		return errors.New(`mail.Send: invalid "bcc" field: ` + err.Error())
	}

	if from == nil || len(to) == 0 {
		return errors.New("mail.Send: must have from and to addresses set")
	}

	var b = new(bytes.Buffer)
	e.Header.Write(b)
	b.WriteString("\r\n\r\n")
	b.Write(e.Message)

	to = append(to, cc...)
	to = append(to, bcc...)

	return e.Mailer(host, e.Auth, from.String(), to.Strings(), b.Bytes())
}
