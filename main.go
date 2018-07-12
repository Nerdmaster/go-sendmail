package main

import (
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"regexp"

	"github.com/Nerdmaster/sendmail/email"
	"github.com/go-yaml/yaml"
	flags "github.com/jessevdk/go-flags"
)

type authorization struct {
	From        string
	To          string
	FromRegex   string `yaml:"from_regex"`
	RewriteFrom string `yaml:"rewrite_from"`
	Catchall    bool
	Username    string
	Password    string
	Host        string
}

type config struct {
	Auths []authorization
	Host  string
}

var opts struct {
	From    string `short:"f" description:"From address"`
	Verbose bool   `short:"v" description:"Verbose mode"`
}

func main() {
	var conf = readConfig()
	var e, err = email.Read(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read stdin: %s", err)
	}
	getCLIArgs(e)

	for _, auth := range conf.Auths {
		if auth.matches(e) {
			err = auth.assignToEmail(e)
			if err != nil {
				log.Fatalf("Unable to assign auth to email: %s", err)
			}
			break
		}
	}

	// Try to send it
	if opts.Verbose {
		log.Printf("DEBUG: trying to send email from %q to %v, message follows", e.From, e.To)
		log.Println(e.Message)
	}

	err = e.Send(conf.Host)
	if err != nil {
		log.Fatalf("Unable to send email (from %q, to %v, msg %q): %s", e.From, e.To, e.Message, err)
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

func (auth *authorization) matches(e *email.Email) bool {
	return auth.Catchall ||
		auth.matchesFrom(e) ||
		auth.matchesFromRegex(e) ||
		auth.matchesTo(e)
}

func (auth *authorization) matchesFrom(e *email.Email) bool {
	return e.From != nil && auth.From == e.From.Address
}

func (auth *authorization) matchesFromRegex(e *email.Email) bool {
	if e.From == nil || auth.FromRegex == "" {
		return false
	}

	var fromRegex, err = regexp.Compile(auth.FromRegex)
	if err != nil {
		log.Fatalf("Invalid regex %q: %s", e.From.Address, err)
	}
	return fromRegex.MatchString(e.From.Address)
}

func (auth *authorization) matchesTo(e *email.Email) bool {
	return len(e.To) > 0 && e.To[0] != nil && auth.To == e.To[0].Address
}

func (auth *authorization) assignToEmail(e *email.Email) error {
	e.Auth = smtp.PlainAuth("", auth.Username, auth.Password, auth.Host)
	if auth.RewriteFrom != "" {
		return e.SetFromAddress(auth.RewriteFrom)
	}
	return nil
}
