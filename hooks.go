package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/YaleSpinup/reaper/common"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Event is the data for an event
type Event struct {
	Action string `json:"action"`
	ID     string `json:"server_id"`
}

// Webhook is the configuration for a webhook
type Webhook struct {
	Client   HTTPClient
	Endpoint string
	Token    string
	Method   string
	Actions  []string
}

// NewWebhook returns a new webhook configuration
func NewWebhook(wh common.Webhook) (Webhook, error) {
	return Webhook{
		Client: &http.Client{
			Timeout: time.Second * 10,
		},
		Endpoint: wh.Endpoint,
		Token:    wh.Token,
		Method:   wh.Method,
		Actions:  wh.Actions,
	}, nil
}

// Send sends a webhook. If the hook is configured as a GET, or HEAD, the hook
// data will be sent as URL parameters.  If the hook is configured as a POST, PUT or PATCH, the data
// will be JSON encoded and sent in the body.
func (wh Webhook) Send(ctx context.Context, event *Event) error {
	for _, e := range wh.Actions {
		if event.Action == e {
			log.Infof("sending webhook event %+v with %+v", event, wh)
			switch wh.Method {
			case http.MethodGet, http.MethodHead:
				url := fmt.Sprintf("%s?id=%s&action=%s", wh.Endpoint, event.ID, event.Action)
				req, err := http.NewRequestWithContext(ctx, wh.Method, url, nil)
				if err != nil {
					return err
				}

				// Add X-Auth-Token if it exists
				if wh.Token != "" {
					req.Header.Add("X-Auth-Token", wh.Token)
				}

				res, err := wh.Client.Do(req)
				if err != nil {
					return err
				}
				defer res.Body.Close()

				log.Debugf("got response from webhook: %+v", res)

				if res.StatusCode > 299 {
					resBody, _ := ioutil.ReadAll(res.Body)
					msg := fmt.Sprintf("Received non-success from webhook (%s) %s(%d %s", url, res.Status, res.StatusCode, resBody)
					return errors.New(msg)
				}
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				body, err := json.Marshal(event)
				if err != nil {
					return errors.Wrap(err, "failed to marshal event into json")
				}
				req, err := http.NewRequestWithContext(ctx, wh.Method, wh.Endpoint, bytes.NewReader(body))
				if err != nil {
					return err
				}

				// Add X-Auth-Token if it exists
				if wh.Token != "" {
					req.Header.Add("X-Auth-Token", wh.Token)
				}

				res, err := wh.Client.Do(req)
				if err != nil {
					return err
				}
				defer res.Body.Close()

				log.Debugf("got response from webhook: %+v", res)
				if res.StatusCode > 299 {
					resBody, _ := ioutil.ReadAll(res.Body)
					msg := fmt.Sprintf("Received non-success from webhook (%s) %s (%d) %s", wh.Endpoint, res.Status, res.StatusCode, resBody)
					return errors.New(msg)
				}
			default:
				log.Warnf("unsupported http method for sending webhook: %s, skipping", wh.Method)
			}
			return nil
		}
	}

	log.Debugf("not sending webhook for event %+v, no matching action", event)
	return nil
}
