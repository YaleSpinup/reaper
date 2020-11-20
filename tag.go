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

// Tagger is the data object for tagging
type Tagger struct {
	Endpoint   string
	Token      string
	ResourceID string
	Org        string
	Client     HTTPClient
}

// NewTagger creates a new tagging object
func NewTagger(endpoint, token, id, org string, encryptToken bool) (Tagger, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	t := token
	if encryptToken {
		crytpT, err := bcrypt.GenerateFromPassword([]byte(token), 4)
		if err != nil {
			return Tagger{}, fmt.Errorf("failed to bcrypt from password: %s", err)
		}

		t = string(crytpT)
	}

	return Tagger{
		Endpoint:   endpoint,
		Token:      t,
		ResourceID: id,
		Org:        org,
		Client:     client,
	}, nil
}

// Tag updates the tags
func (t Tagger) Tag(tags map[string]string) error {
	log.Debugf("Tagging with endpoint: %s, resource: %s org: %s and tags %+v", t.Endpoint, t.ResourceID, t.Org, tags)

	data, err := json.Marshal(struct {
		Tags map[string]string `json:"tags"`
	}{
		Tags: tags,
	})
	if err != nil {
		return err
	}

	log.Debugf("Marshalled JSON body %s, creating new HTTP request", string(data))

	url := fmt.Sprintf("%s/%s/%s/tags", t.Endpoint, t.Org, t.ResourceID)
	log.Debugf("Generated URL for tag request: %s", url)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Forwarded-User", "reaper")
	req.Header.Set("X-Auth-token", t.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := t.Client.Do(req)
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
