package eventreporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type SlackReporter struct {
	Endpoint string
	Token    string
	Channel  string
	Username string
	Icon     string
	Client   HTTPClient
}

func NewSlackReporter(options map[string]string) (*SlackReporter, error) {
	var r SlackReporter
	if _, ok := options["endpoint"]; !ok {
		return &r, errors.New("Slack endpoint must be specified")
	}
	r.Endpoint = options["endpoint"]

	if c, ok := options["channel"]; !ok {
		r.Channel = "#general"
	} else {
		r.Channel = c
	}

	if u, ok := options["username"]; !ok {
		r.Username = "SlackReporter"
	} else {
		r.Username = u
	}

	if i, ok := options["icon"]; !ok {
		r.Icon = ":speak_no_evil:"
	} else {
		r.Icon = i
	}

	r.Client = &http.Client{Timeout: 30 * time.Second}
	return &r, nil
}

func (r *SlackReporter) Report(e *Event) error {
	log.Infof("Reporting event via Slack: %s", e.Message)

	data, err := json.Marshal(struct {
		Text      string `json:"text"`
		Channel   string `json:"channel"`
		LinkNames int    `json:"link_names"`
		Username  string `json:"username"`
		IconEmoji string `json:"icon_emoji"`
	}{
		Text:      e.Message,
		Channel:   r.Channel,
		LinkNames: 1,
		Username:  r.Username,
		IconEmoji: r.Icon,
	})
	if err != nil {
		return err
	}

	log.Debugf("Marshalled JSON body %s, creating new HTTP request", string(data))
	req, err := http.NewRequest("POST", r.Endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	log.Debugln("Got response from POST:", res.StatusCode)

	if res.StatusCode > 299 {
		return fmt.Errorf("Got a non-success http response from http POST to %s, %d", r.Endpoint, res.StatusCode)
	}

	return nil
}
