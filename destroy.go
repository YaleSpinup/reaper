package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Destroyer is the data object for destruction
type Destroyer struct {
	Endpoint   string
	Token      string
	ResourceID string
	Org        string
	Client     HTTPClient
}

// NewDestroyer creates a new destruction object
func NewDestroyer(endpoint, token, id, org string) Destroyer {
	client := &http.Client{Timeout: 30 * time.Second}
	return Destroyer{
		Endpoint:   endpoint,
		Token:      token,
		ResourceID: id,
		Org:        org,
		Client:     client,
	}
}

// Destroy destroys the instance by 'DELETE'ing it
func (d Destroyer) Destroy() error {
	log.Debugf("Destroying with endpoint: %s, resource: %s, org: %s  ", d.Endpoint, d.ResourceID, d.Org)

	url := fmt.Sprintf("%s/%s/%s", d.Endpoint, d.Org, d.ResourceID)
	log.Debugf("Generated URL for delete request: %s", url)

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}

	req.Header.Set("X-Forwarded-User", "reaper")
	req.Header.Set("Auth-token", d.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := d.Client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	if res.StatusCode > 299 {
		return fmt.Errorf("Got a non-success http response from http DELETE to %s, %d", url, res.StatusCode)
	}

	return nil
}
