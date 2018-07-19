package email

import (
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

// rwHeader is internal to let us manipulate the header directly
type rwHeader struct {
	Header
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
func (h rwHeader) Set(key, value string) {
	textproto.MIMEHeader(h.h).Set(key, value)
}

// Del removes the given header
func (h rwHeader) Del(key string) {
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
	From    *mail.Address
	To      AddressList
	CC      AddressList
	BCC     AddressList
	Message []byte
	header  rwHeader
	Auth    smtp.Auth
	Mailer  func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// New returns a basic Email instance with its Mailer set to the default smtp.SendMail
func New() *Email {
	return &Email{Mailer: smtp.SendMail}
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
	var m, err = mail.ReadMessage(r)
	if err != nil {
		return err
	}

	e.header.h = m.Header
	e.Message, err = ioutil.ReadAll(m.Body)
	if err != nil {
		return err
	}

	return e.parseHeader()
}

// parseHeader finds the email-related header for from/to/cc/bcc so we can
// properly populate the smtp send
func (e *Email) parseHeader() error {
	var from *mail.Address
	var list AddressList
	var err error

	from, err = e.header.Address("from")
	if from != nil {
		e.From = from
	}

	if err == nil {
		list, err = e.header.AddressList("to")
		if len(list) > 0 {
			e.To = list
		}
	}

	if err == nil {
		list, err = e.header.AddressList("cc")
		if len(list) > 0 {
			e.CC = list
		}
	}

	if err == nil {
		list, err = e.header.AddressList("bcc")
		if len(list) > 0 {
			e.BCC = list
		}
	}

	return err
}

// SetFromAddress parses addr into a mail.Address, returning an error if
// parsing fails.  Replaces the existing From address if it exists.
func (e *Email) SetFromAddress(addr string) error {
	var from, err = mail.ParseAddress(addr)
	if err == nil {
		e.From = from
	}
	return err
}

// SetToAddresses parses addrlist into a list of mail.Addresses which replaces
// the current To value, returning an error if parsing fails.
func (e *Email) SetToAddresses(addrlist string) error {
	var tolist, err = mail.ParseAddressList(addrlist)
	if err == nil {
		e.To = tolist
	}
	return err
}

// Header returns a read-only copy of the email header after ensuring it reflects all
// settable values, such as when e.From is assigned manually or
// e.SetToAddresses is called.  Affects all currently modifiable fields: From,
// To, CC, and BCC.
func (e *Email) Header() Header {
	var h = rwHeader{e.header.Clone()}
	h.Del("from")
	h.Del("to")
	h.Del("cc")
	h.Del("bcc")

	if e.From != nil {
		h.Set("from", e.From.String())
	}
	if len(e.To) > 0 {
		h.Set("to", e.To.String())
	}
	if len(e.CC) > 0 {
		h.Set("cc", e.CC.String())
	}
	if len(e.BCC) > 0 {
		h.Set("bcc", e.BCC.String())
	}

	return h.Header
}

// Send uses the from/to/message data combined with host to attempt to send the
// message via smtp
func (e *Email) Send(host string) error {
	if e.From == nil || len(e.To) == 0 {
		return errors.New("mail.Send: must have from and to addresses set")
	}

	var b = new(bytes.Buffer)
	e.Header().Write(b)
	b.WriteString("\r\n\r\n")
	b.Write(e.Message)

	var addrs AddressList
	addrs = append(addrs, e.To...)
	addrs = append(addrs, e.CC...)
	addrs = append(addrs, e.BCC...)

	var err = e.Mailer(host, e.Auth, e.From.String(), addrs.Strings(), b.Bytes())
	return err
}
