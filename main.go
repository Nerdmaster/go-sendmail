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

func main() {
	var conf = readConfig()
	var e = email.New()

	getCLIArgs(e)
	var err = e.Read(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read stdin: %s", err)
	}
	if e.From != nil {
		e.Auth = getAuth(conf, e.From.Address)
	}

	// Try to send it
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

var opts struct {
	From string `short:"f" description:"From address"`
}

func getCLIArgs(e *email.Email) {
	var args, err = flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}
	if opts.From != "" {
		err = e.SetFromAddress(opts.From)
		if err != nil {
			log.Fatalf(`Unable to set "from" address: %s`, err)
		}
	}

	for _, arg := range args {
		err = e.AddToAddresses(arg)
		if err != nil {
			log.Fatalf(`Unable to set "to" address: %s`, err)
		}
	}
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
