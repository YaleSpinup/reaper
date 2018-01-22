package eventreporter

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailReporter wraps up the configuration needed for sending email events
type EmailReporter struct {
	Address string
	Auth    smtp.Auth
	To      []string
	From    string
}

// NewEmailReporter takes a map of options and produces a reporter that can
// send email message events
func NewEmailReporter(options map[string]string) (*EmailReporter, error) {
	var e EmailReporter

	if m, ok := options["mailserver"]; !ok {
		return &e, errors.New("A mail server endpoint must be specified")
	} else if len(strings.Split(m, ":")) != 2 {
		msg := fmt.Sprintf("The given mail server value (%s) seems invalid, it should be the form foo.bar.com:25", m)
		return &e, errors.New(msg)
	}
	e.Address = options["mailserver"]

	var auth smtp.Auth
	if u, ok := options["username"]; ok {
		if p, ok := options["password"]; ok {
			server := strings.Split(e.Address, ":")
			auth = smtp.PlainAuth("", u, p, server[0])
		} else if s, ok := options["secret"]; ok {
			auth = smtp.CRAMMD5Auth(u, s)
		}
	}
	e.Auth = auth

	if from, ok := options["from"]; !ok {
		return &e, errors.New("An email from address must be specified")
	} else if len(strings.Split(from, "@")) != 2 {
		msg := fmt.Sprintf("The given email from address (%s) seems invalid, it should be the form foo@bar.com", from)
		return &e, errors.New(msg)
	}
	e.From = options["from"]

	if _, ok := options["to"]; !ok {
		return &e, errors.New("At least one recipient address must be specified")
	}
	e.To = strings.Split(options["to"], ";")

	return &e, nil
}

// Report sends an email event
func (r *EmailReporter) Report(e Event) error {
	return smtp.SendMail(r.Address, r.Auth, r.From, r.To, []byte(e.Message))
}
