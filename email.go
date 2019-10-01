package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var warningTemplate = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
  <head></head>
  <body>
    <p>
      Hello {{if .FirstName}}{{ .FirstName }}{{else}}{{ .NetID }}{{end}},
    </p>
    <p>
      Your Spinup TryIT server {{.FQDN}} will expire on {{.ExpireOn}}. If you would like to keep it, please renew it from the Spinup interface or by clicking the following link (this e-mail's link is one-time use):
      <br />
      <br />
      <a href="{{.RenewalLink}}">{{.RenewalLink}}</a>
    </p>
    <p>
      Cheers,<br />
			Spinup Team<br />
			<a href="{{.SpinupURL}}">{{.SpinupURL}}</a>
			<a href="{{.SpinupSiteURL}}">{{.SpinupSiteURL}}</a>
    </p>
  </body>
</html>
`

var decomTemplate = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
  <head></head>
  <body>
    <p>
      Hello {{if .FirstName}}{{ .FirstName }}{{else}}{{ .NetID }}{{end}},
    </p>
    <p>
      Your Spinup TryIT server {{.FQDN}} expired on {{.ExpireOn}} and has been deleted.  Thank you for using Spinup TryIT!
    </p>
    <p>
      Cheers,<br />
			Spinup Team<br />
			<a href="{{.SpinupURL}}">{{.SpinupURL}}</a>
			<a href="{{.SpinupSiteURL}}">{{.SpinupSiteURL}}</a>
    </p>
  </body>
</html>
`

var renewalTemplate = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
  <head></head>
    <body>
      <p>
        Hello {{if .FirstName}}{{ .FirstName }}{{else}}{{ .NetID }}{{end}},
      </p>
      <p>
        Your Spinup TryIT server {{.FQDN}} has been renewed and will expire on {{.ExpireOn}}.  Thank you for using Spinup TryIT!
      </p>
      <p>
        Cheers,<br />
				Spinup Team<br />
				<a href="{{.SpinupURL}}">{{.SpinupURL}}</a>
				<a href="{{.SpinupSiteURL}}">{{.SpinupSiteURL}}</a>
      </p>
    </body>
</html>
`

// SendMail sends an email with plain auth
func SendMail(address, body, from, password, subject, to, username string) error {
	if len(strings.Split(address, ":")) != 2 {
		return fmt.Errorf("The given mail server value (%s) seems invalid, it should be the form foo.bar.com:25", address)
	}

	if len(strings.Split(from, "@")) != 2 {
		return fmt.Errorf("The given email from address (%s) seems invalid, it should be the form foo@bar.com", from)
	}

	// message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	message := "From: " + from + "\nTo: " + to + "\nSubject: " + subject + "\n" + mime + "\n" + body

	log.Debugf("Sending mail to %s from %s via %s with body\n%s", to, from, address, message)

	var auth smtp.Auth
	if username == "" || password == "" {
		log.Info("Using nil auth for SMTP")
		auth = nil
	} else {
		server := strings.Split(address, ":")
		auth = smtp.PlainAuth("", username, password, server[0])
	}

	// Note that I'm being lazy here and only allowing one recipient #quickwins
	return smtp.SendMail(address, auth, from, []string{to}, []byte(message))
}

// ParseWarningTemplate takes a map of parameters and parses the warning template, returning the parsed string
func ParseWarningTemplate(params map[string]string) (string, error) {
	tmpl, err := template.New("warningTemplate").Parse(warningTemplate)
	if err != nil {
		return "", err
	}

	buffer := new(bytes.Buffer)
	err = tmpl.Execute(buffer, struct {
		ExpireOn      string
		FirstName     string
		NetID         string
		FQDN          string
		RenewalLink   string
		SpinupURL     string
		SpinupSiteURL string
	}{
		ExpireOn:      params["expire_on"],
		FirstName:     params["first"],
		NetID:         params["netid"],
		FQDN:          params["fqdn"],
		RenewalLink:   params["link"],
		SpinupURL:     params["spinupURL"],
		SpinupSiteURL: params["spinupSiteURL"],
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// ParseRenewalTemplate takes a map of parameters and parses the renewal template, returning the parsed string
func ParseRenewalTemplate(params map[string]string) (string, error) {
	tmpl, err := template.New("renewalTemplate").Parse(renewalTemplate)
	if err != nil {
		return "", err
	}

	buffer := new(bytes.Buffer)
	err = tmpl.Execute(buffer, struct {
		ExpireOn      string
		FirstName     string
		NetID         string
		FQDN          string
		SpinupURL     string
		SpinupSiteURL string
	}{
		ExpireOn:      params["expire_on"],
		FirstName:     params["first"],
		NetID:         params["netid"],
		FQDN:          params["fqdn"],
		SpinupURL:     params["spinupURL"],
		SpinupSiteURL: params["spinupSiteURL"],
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// ParseDecomTemplate takes a map of parameters and parses the decom template, returning the parsed string
func ParseDecomTemplate(params map[string]string) (string, error) {
	tmpl, err := template.New("decomTemplate").Parse(decomTemplate)
	if err != nil {
		return "", err
	}

	buffer := new(bytes.Buffer)
	err = tmpl.Execute(buffer, struct {
		ExpireOn      string
		FirstName     string
		NetID         string
		FQDN          string
		SpinupURL     string
		SpinupSiteURL string
	}{
		ExpireOn:      params["expire_on"],
		FirstName:     params["first"],
		NetID:         params["netid"],
		FQDN:          params["fqdn"],
		SpinupURL:     params["spinupURL"],
		SpinupSiteURL: params["spinupSiteURL"],
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}
