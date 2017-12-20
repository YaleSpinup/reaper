package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Notifier is the data object for notifying
type Notifier struct {
	Endpoint string
	Token    string
	Client   HTTPClient
}

// NewNotifier creates a new notification object with the given params
func NewNotifier(endpoint, token string) Notifier {
	client := &http.Client{Timeout: 30 * time.Second}
	return Notifier{
		Endpoint: endpoint,
		Token:    token,
		Client:   client,
	}
}

// Notify sends the notification
func (n Notifier) Notify(params map[string]string) error {
	log.Debugf("Notifying with endpoint: %s, and params: %+v", n.Endpoint, params)

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	log.Debugf("Marshalled JSON body %s, creating new HTTP request", string(data))

	req, err := http.NewRequest(http.MethodPost, n.Endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Auth-token", n.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := n.Client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Warnf("Couldn't read response body: %s", err.Error())
	}

	log.Debugf("Got %d response from %s with body: %s", res.StatusCode, n.Endpoint, body)

	if res.StatusCode > 299 {
		return fmt.Errorf("Got a non-success http response from http POST to %s, %d", n.Endpoint, res.StatusCode)
	}

	return nil
}
