package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNewUserFetcher(t *testing.T) {
	restUserFetcher := map[string]string{
		"type":     "rest",
		"endpoint": "http://127.0.0.1:1234/api/user",
		"token":    "sekret",
	}

	r, err := NewUserFetcher(restUserFetcher)
	if err != nil {
		t.Error("Error generating new RESTUserFetcher from", restUserFetcher)
	}

	rType := reflect.TypeOf(r).String()
	if string(rType) != "*main.RESTUserFetcher" {
		t.Errorf("NewUserFetcher returned the wrong type.  Expected: RESTUserFetcher, got %s", rType)
	}
}

func TestRESTUserFetcherConfigure(t *testing.T) {
	expectedRESTUserFetcher := RESTUserFetcher{
		Endpoint: "http://127.0.0.1:1234/api/user",
		Token:    "sekret",
		Client:   nil,
	}

	config := map[string]string{
		"type":     "rest",
		"endpoint": "http://127.0.0.1:1234/api/user",
		"token":    "sekret",
	}

	var actualRESTUserFetcher RESTUserFetcher
	err := actualRESTUserFetcher.Configure(config)
	if err != nil {
		t.Errorf("Failed to configure REST User Fetcher: %s", err)
	}
	actualRESTUserFetcher.Client = nil

	if !reflect.DeepEqual(expectedRESTUserFetcher, actualRESTUserFetcher) {
		t.Errorf("Expected RESTUserFetcher Configure to return %+v, got %+v", expectedRESTUserFetcher, actualRESTUserFetcher)
	}
}

func TestRESTUserFetcherFetch(t *testing.T) {
	expectedUser := &User{
		First: "Focal",
		Last:  "Banger",
		Email: "focal.banger@alchemist.com",
		NetID: "fb4me",
	}

	b, _ := json.Marshal(expectedUser)
	client := NewMockClient(b, 200)
	restUserFetcher := RESTUserFetcher{
		Endpoint: "http://127.0.0.1:1234/api/user",
		Token:    "sekret",
		Client:   client,
	}

	actualUser, err := restUserFetcher.FetchByID("fb4me")
	if err != nil {
		t.Errorf("Failed to fetch a user frmo a REST User Fetcher")
	}

	if !reflect.DeepEqual(expectedUser, actualUser) {
		t.Errorf("Expected RESTUserFetcher to return a user %+v, got %+v", expectedUser, actualUser)
	}
}
