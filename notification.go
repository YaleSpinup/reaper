package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Notifier is the data object for notifying
type Notifier struct {
	Endpoint string
	Token    string
	Params   map[string]string
}

// NewNotifier creates a new notification object with the given params
func NewNotifier(endpoint, token string, params map[string]string) *Notifier {
	return &Notifier{
		Endpoint: endpoint,
		Token:    token,
		Params:   params,
	}
}

// Notify sends the notification
func (n *Notifier) Notify() error {
	log.Debugf("Notifying with %+v", *n)

	client := &http.Client{Timeout: 30 * time.Second}
	data, err := json.Marshal(n.Params)
	if err != nil {
		return err
	}

	log.Debugf("Marshalled JSON body %s, creating new HTTP request", string(data))

	req, err := http.NewRequest("POST", n.Endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Auth-Token", n.Token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode > 299 {
		return fmt.Errorf("Got a non-success http response from http POST to %s, %d", n.Endpoint, res.StatusCode)
	}

	return nil
}
