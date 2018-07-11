package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"net/smtp"
	"os"
	"regexp"
	"strings"

	"github.com/go-yaml/yaml"
	flags "github.com/jessevdk/go-flags"
)

var lineRegexp = regexp.MustCompile("\r\n|\r|\n")

type authorization struct {
	From      string
	FromRegex string `yaml:"from_regex"`
	Catchall  bool
	Username  string
	Password  string
	Host      string
}

type config struct {
	Auths []authorization
	Host  string
}

type email struct {
	From    *mail.Address
	To      []*mail.Address
	Message string
	Auth    smtp.Auth
}

// setupMessage stores the message and then looks for headers in order to
// determine from/to in case those weren't passed on the command line
func (e *email) setupMessage(message string) {
	e.Message = message
	for _, line := range lineRegexp.Split(message, -1) {
		// The first blank line means we're done with headers, so there's no more data to be gleaned
		if line == "" {
			return
		}

		if e.From == nil && strings.HasPrefix(line, "From: ") {
			e.setFromAddress(line[6:])
		}

		if strings.HasPrefix(line, "Cc: ") {
			e.addToAddresses(line[4:])
		}
		if strings.HasPrefix(line, "Bcc: ") {
			e.addToAddresses(line[5:])
		}
		if strings.HasPrefix(line, "To: ") {
			e.addToAddresses(line[4:])
		}
	}
}

func (e *email) setFromAddress(addr string) {
	var err error
	e.From, err = mail.ParseAddress(addr)
	if err != nil {
		log.Fatalf(`Invalid "from" address %q: %s`, addr, err)
	}
}

func (e *email) addToAddresses(addrlist string) {
	var addrs, err = mail.ParseAddressList(addrlist)
	if err != nil {
		log.Fatalf("Invalid address(es) %q: %s", addrlist, err)
	}
	e.To = append(e.To, addrs...)
}

func main() {
	var conf = readConfig()
	var e = new(email)

	getCLIArgs(e)
	e.setupMessage(parseStdinEmailMessage())
	e.Auth = getAuth(conf, e.From.Address)

	// Try to send it
	var toList = make([]string, len(e.To))
	for i, to := range e.To {
		toList[i] = to.String()
	}
	var err = smtp.SendMail(conf.Host, e.Auth, e.From.String(), toList, []byte(e.Message))
	if err != nil {
		log.Fatalf("Unable to send email (from %q, to %q, msg %q): %s", e.From, e.To, e.Message, err)
	}
}

func readConfig() config {
	var err error
	var fname = "config.yml"
	_, err = os.Stat("config.yml")
	if os.IsNotExist(err) {
		fname = "/etc/go-sendmail.yml"
	}

	var data []byte
	data, err = ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Unable to open %q: %s", fname, err)
	}

	var conf config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalf("Unable to parse yaml: %s", err)
	}

	return conf
}

var opts struct {
	From string `short:"f" description:"From address"`
}

func getCLIArgs(e *email) {
	var args, err = flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}
	if opts.From != "" {
		e.setFromAddress(opts.From)
	}

	for _, arg := range args {
		e.addToAddresses(arg)
	}
}

func parseStdinEmailMessage() string {
	var eof bool
	var rawMessage []byte
	var message string
	var buf [10240]byte
	for !eof {
		var xbuf = buf[0:]
		var n, err = os.Stdin.Read(xbuf)
		if err != nil && err != io.EOF {
			log.Fatalf("Error reading from stdin: %s", err)
		}

		rawMessage = append(rawMessage, xbuf[:n]...)
		message, eof = bytesToMessage(rawMessage)
		if eof || err == io.EOF {
			return message
		}
	}

	return ""
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

func makeAuth(auth authorization) smtp.Auth {
	return smtp.PlainAuth("", auth.Username, auth.Password, auth.Host)
}

// getAuth reads through all the auths in the config and uses the first which matches
func getAuth(conf config, from string) smtp.Auth {
	for _, auth := range conf.Auths {
		if auth.From == from {
			return makeAuth(auth)
		}

		if auth.FromRegex != "" {
			var fromRegex, err = regexp.Compile(auth.FromRegex)
			if err != nil {
				log.Fatalf("Invalid regex %q: %s", fromRegex, err)
			}
			if fromRegex.MatchString(from) {
				return makeAuth(auth)
			}
		}

		if auth.Catchall {
			return makeAuth(auth)
		}
	}

	return nil
}
