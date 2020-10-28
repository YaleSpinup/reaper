package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Decommissioner is the data object for decommissioning
type Decommissioner struct {
	Endpoint     string
	Token        string
	EncryptToken bool
	ResourceID   string
	Org          string
	Client       HTTPClient
}

// NewDecommissioner creates a new decommissioning object
func NewDecommissioner(endpoint, token, id, org string, encryptToken bool) (Decommissioner, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	t := token
	if encryptToken {
		crytpT, err := bcrypt.GenerateFromPassword([]byte(token), 4)
		if err != nil {
			return Decommissioner{}, fmt.Errorf("failed to bcyrpt from password: %s", err)
		}

		t = string(crytpT)
	}

	return Decommissioner{
		Endpoint:   endpoint,
		Token:      t,
		ResourceID: id,
		Org:        org,
		Client:     client,
	}, nil
}

// SetStatus decommissions the instance by 'PUT'ing a new status to it
func (d Decommissioner) SetStatus() error {
	log.Debugf("Decomming with endpoint: %s, resource: %s, org: %s  ", d.Endpoint, d.ResourceID, d.Org)

	data, err := json.Marshal(struct {
		Status string `json:"status"`
	}{
		Status: "decom",
	})
	if err != nil {
		return err
	}

	log.Debugf("Marshalled JSON body %s, creating new HTTP request", string(data))

	url := fmt.Sprintf("%s/%s/%s/status", d.Endpoint, d.Org, d.ResourceID)
	log.Debugf("Generated URL for decom status request: %s", url)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Forwarded-User", "reaper")
	req.Header.Set("Auth-token", d.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := d.Client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	if res.StatusCode > 299 {
		return fmt.Errorf("Got a non-success http response from http PUT to %s, %d", url, res.StatusCode)
	}

	return nil
}
