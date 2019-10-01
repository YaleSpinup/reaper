package common

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	BaseURL          string
	Decommission     Decommissioner
	Destroy          Destroyer
	Email            Emailer
	Filter           map[string]string
	Interval         string
	Listen           string
	LogLevel         string
	Notify           Notifier
	SearchEngine     map[string]string
	UserDatasource   map[string]string
	Tagging          Tagging
	EncryptionSecret string
	RedirectURL      string
	SpinupURL        string
	SpinupSiteURL    string
	Token            string
	EventReporters   map[string]map[string]string
	Webhooks         []Webhook
}

// Emailer configures the email sending process
type Emailer struct {
	From       string
	Mailserver string
	Password   string
	Username   string
}

// Notifier configures the notification process
type Notifier struct {
	Age      []string
	Endpoint string
	Token    string
}

// Destroyer configures the deletion process
type Destroyer struct {
	Age      string
	Endpoint string
	Token    string
}

// Decommissioner configures the decom process
type Decommissioner struct {
	Age      string
	Endpoint string
	Token    string
}

// Tagging configures the tag update process
type Tagging struct {
	Endpoint string
	Token    string
}

// Webhook configures the webhook endpoints
type Webhook struct {
	Endpoint string
	Token    string
	Method   string
	Actions  []string
}

// ReadConfig decodes the configuration from an io Reader
func ReadConfig(r io.Reader) (Config, error) {
	var c Config
	log.Infoln("Reading configuration")
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return c, errors.Wrap(err, "unable to decode JSON message")
	}
	return c, nil
}
