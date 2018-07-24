package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/mail"
	"net/smtp"
	"os"

	"github.com/Nerdmaster/sendmail/email"
	"github.com/go-yaml/yaml"
	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	From    string `short:"f" description:"From address"`
	Dryrun  bool   `short:"n" description:"Dry run; do not send an email message"`
	Verbose bool   `short:"v" description:"Verbose mode"`
}

func fatalWithEmail(e *email.Email, err error) {
	var from = e.Header.Get("from")
	var to = e.Header.Get("to")
	log.Fatalf("Unable to send email (from %q, to %v, msg %q): %s", from, to, e.Message, err)
}

func main() {
	var args, err = flags.Parse(&opts)
	if err != nil {
		log.Fatalf("Unable to parse CLI flags: %s", err)
	}

	var rules = readRules()
	if len(rules) == 0 {
		log.Fatalf("No rules configured")
	}
	var e *email.Email
	e, err = email.Read(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read stdin: %s", err)
	}
	applyArgs(e, args)

	var matchFound bool
	for _, rule := range rules {
		if process(rule, e) {
			matchFound = true
			break
		}
	}

	if !matchFound {
		fatalWithEmail(e, errors.New("no rules matched"))
	}
}

// process tries to match the rule against the email, setting up auth and
// sending the message if it matches.  Returns whether processing occurred.
func process(r *RuleConf, e *email.Email) bool {
	if !r.rule.Match(e) {
		return false
	}

	if opts.Verbose {
		log.Printf("DEBUG: Matched rule (matchers: %#v)", r.Matchers)
	}

	var a = r.Auth
	e.Auth = smtp.PlainAuth("", a.Username, a.Password, a.Host)

	// Try to send it
	if opts.Verbose {
		log.Printf("DEBUG: trying to send email from %q to %v, message follows",
			e.Header.Get("from"), e.Header.Get("to"))
		log.Println(string(e.Message))
	}

	if opts.Dryrun {
		log.Printf("Dry run requested; not sending email")
	} else {
		var err = e.Send(a.Server)
		if err != nil {
			fatalWithEmail(e, err)
		}
	}

	return true
}

func readRules() []*RuleConf {
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

	var rlist []*RuleConf
	err = yaml.Unmarshal(data, &rlist)
	if err != nil {
		log.Fatalf("Unable to parse yaml: %s", err)
	}

	initRules(rlist)
	return rlist
}

func applyArgs(e *email.Email, args []string) {
	if opts.From != "" {
		var from, err = mail.ParseAddress(opts.From)
		if err != nil {
			log.Fatalf(`Unable to set "from" address %q: %s`, opts.From, err)
		}
		e.Header.Set("from", from.String())
	}

	var tolist email.AddressList
	for _, arg := range args {
		var to, err = mail.ParseAddress(arg)
		if err != nil {
			log.Fatalf(`Unable to set "to" address %q: %s`, arg, err)
		}
		tolist = append(tolist, to)
	}
	if len(tolist) > 0 {
		e.Header.Set("to", tolist.String())
	}
}
