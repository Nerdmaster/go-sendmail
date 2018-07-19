package main

import (
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

func main() {
	var conf = readConfig()
	var e, err = email.Read(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read stdin: %s", err)
	}
	getCLIArgs(e)

	for _, rule := range conf.Rules {
		if rule.Match(e) {
			process(e, rule.Auth)
			break
		}
	}
}

func process(e *email.Email, a *rule.Authentication) {
	e.Auth = smtp.PlainAuth("", a.Username, a.Password, a.Host)

	// Try to send it
	if opts.Verbose {
		log.Printf("DEBUG: trying to send email from %q to %v, message follows", e.From, e.To)
		log.Println(e.Message)
	}

	if !opts.Dryrun {
		var err = e.Send(a.Server)
		if err != nil {
			log.Fatalf("Unable to send email (from %q, to %v, msg %q): %s", e.From, e.To, e.Message, err)
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
