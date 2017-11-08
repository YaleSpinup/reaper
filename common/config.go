package common

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	Listen       string
	SearchEngine map[string]string
	Filter       map[string]string
	LogLevel     string
	Orgs         []string
	Interval     string
	Notify       Notifier
	Decommission Decommissioner
	Destroy      Destroyer
	Token        string
}

// Notifier configures the notification process
type Notifier struct {
	Age      []string
	Endpoint string
	Token    string
}

// Decommissioner configures the decom process
type Decommissioner struct {
	Age      string
	Endpoint string
	Token    string
}

// Destroyer configures the deletion process
type Destroyer struct {
	Age      string
	Endpoint string
	Token    string
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
