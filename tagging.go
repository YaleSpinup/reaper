package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Tagger is the data object for tagging
type Tagger struct {
	Endpoint   string
	Token      string
	ResourceID string
	Org        string
	Tags       map[string]string
}

// NewTagger creates a new tagging object
func NewTagger(endpoint, token, id, org string, tags map[string]string) *Tagger {
	return &Tagger{
		Endpoint:   endpoint,
		Token:      token,
		ResourceID: id,
		Org:        org,
		Tags:       tags,
	}
}

// Tag updates the tags
func (t *Tagger) Tag() error {
	log.Debugf("Tagging with %+v", *t)

	client := &http.Client{Timeout: 30 * time.Second}
	data, err := json.Marshal(struct {
		Tags map[string]string `json:"tags"`
	}{
		Tags: t.Tags,
	})
	if err != nil {
		return err
	}

	log.Debugf("Marshalled JSON body %s, creating new HTTP request", string(data))

	url := fmt.Sprintf("%s/%s/%s/tags", t.Endpoint, t.Org, t.ResourceID)
	log.Debugf("Generated URL for tag request: %s", url)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Auth-Token", t.Token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode > 299 {
		return fmt.Errorf("Got a non-success http response from http PUT to %s, %d", url, res.StatusCode)
	}

	return nil
}
