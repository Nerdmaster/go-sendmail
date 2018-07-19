package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"

	"github.com/Nerdmaster/sendmail/email"
	"github.com/Nerdmaster/sendmail/rule"
	"github.com/go-yaml/yaml"
	flags "github.com/jessevdk/go-flags"
)

type config struct {
	Rules []*rule.Rule
}

var opts struct {
	From    string `short:"f" description:"From address"`
	Dryrun  bool   `short:"n" description:"Dry run; do not send an email message"`
	Verbose bool   `short:"v" description:"Verbose mode"`
}

func fatalWithEmail(e *email.Email, err error) {
	log.Fatalf("Unable to send email (from %q, to %v, msg %q): %s", e.From, e.To, e.Message, err)
}

func main() {
	var conf = readConfig()
	if len(conf.Rules) == 0 {
		log.Fatalf("No rules configured")
	}
	var e, err = email.Read(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read stdin: %s", err)
	}
	getCLIArgs(e)

	var matchFound bool
	for i, rule := range conf.Rules {
		if rule.Match(e) {
			if opts.Verbose {
				log.Printf("DEBUG: Matched rule %d (matchers: %#v)", i+1, rule.Matchers)
			}
			process(e, rule.Auth)
			matchFound = true
			break
		}
	}

	if !matchFound {
		fatalWithEmail(e, errors.New("no rules matched"))
	}
}

func process(e *email.Email, a *rule.Authentication) {
	e.Auth = smtp.PlainAuth("", a.Username, a.Password, a.Host)

	// Try to send it
	if opts.Verbose {
		log.Printf("DEBUG: trying to send email from %q to %v, message follows", e.From, e.To)
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

func getCLIArgs(e *email.Email) {
	var args, err = flags.Parse(&opts)
	if err != nil {
		log.Fatalf("Unable to parse CLI flags: %s", err)
	}
	if opts.From != "" {
		err = e.SetFromAddress(opts.From)
		if err != nil {
			log.Fatalf(`Unable to set "from" address: %s`, err)
		}
	}

	for _, arg := range args {
		err = e.SetToAddresses(arg)
		if err != nil {
			log.Fatalf(`Unable to set "to" address: %s`, err)
		}
	}
}
