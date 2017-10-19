package common

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	SearchEngine map[string]string
	Filters      Filter
	LogLevel     string
	Orgs         []string
	Interval     string
	Schedule     Schedule
}

// Filter defines a set of positive matches for things to search
type Filter struct {
	TagKeyValues map[string][]string
}

// Schedule defines the schedule configuration
type Schedule struct {
	Notify string
	Decom  string
	Reap   string
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
