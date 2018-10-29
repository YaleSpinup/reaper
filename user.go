package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// User is a spinup user object from the rest interface
type User struct {
	First string
	Last  string
	Email string
	NetID string
}

// UserFetcher defines an interface for getting user details
type UserFetcher interface {
	FetchByID(id string) (*User, error)
	Configure(config map[string]string) error
}

// NewUserFetcher creates a new user fetcher and configures it
func NewUserFetcher(config map[string]string) (UserFetcher, error) {
	log.Debugf("Creating a new user fetcher with config %+v", config)
	switch config["type"] {
	case "rest":
		u := new(RESTUserFetcher)
		err := u.Configure(config)
		return u, err
	}
	return nil, fmt.Errorf("Couldn't find appropriate provider to create for %s", config["type"])
}

// RESTUserFetcher is the configuration detail for getting a user's details from a REST endpoint
type RESTUserFetcher struct {
	Endpoint string
	Token    string
	Client   HTTPClient
}

// GetUserByID fetches a user by ID
func GetUserByID(f UserFetcher, id string) (*User, error) {
	return f.FetchByID(id)
}

// Configure sets up a new RESTful User Fetcher
func (u *RESTUserFetcher) Configure(config map[string]string) error {
	if _, ok := config["endpoint"]; !ok {
		return fmt.Errorf("Endpoint required and not found in RESTUserFetch configuration")
	}
	u.Endpoint = config["endpoint"]

	if _, ok := config["token"]; !ok {
		return fmt.Errorf("Token required and not found in RESTUserFetch configuration")
	}
	u.Token = config["token"]

	client := &http.Client{Timeout: 30 * time.Second}
	if timeout, ok := config["timeout"]; ok {
		t, err := time.ParseDuration(timeout)
		if err != nil {
			log.Errorf("Invalid timeout specified for RESTUserFetcher: %s", err)
			return err
		}
		client.Timeout = t
	}
	u.Client = client

	return nil
}

// FetchByID gets a user by ID from a REST endpoint
func (u *RESTUserFetcher) FetchByID(id string) (*User, error) {
	url := fmt.Sprintf("%s/%s", u.Endpoint, id)
	log.Debugf("Fetching user id %s with from %s", id, url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Forwarded-User", "reaper")
	req.Header.Set("Auth-token", u.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	log.Debug("HTTP response from REST User Fetch by ID:", res)

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	if res.StatusCode > 299 {
		return nil, fmt.Errorf("Got a non-success http response from http GET to %s, %d", url, res.StatusCode)
	}

	user := new(User)
	err = json.NewDecoder(res.Body).Decode(user)
	if err != nil {
		log.Error("Failed to unmarshall request for user data:", err)
		return nil, err
	}
	return user, nil
}
