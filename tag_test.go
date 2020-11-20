package main

import (
	"net/http"
	"reflect"
	"testing"
)

var (
	testTagEndpoint     = "http://127.0.0.1/v1/servers"
	testTagToken        = "xypdq"
	testTagResourceID   = "i-abc123"
	testTagOrg          = "complaintdept"
	testTagEncryptToken = false
	testTagClient       = NewMockClient([]byte("ok"), 200)
	testTagger          = Tagger{
		Endpoint:   testTagEndpoint,
		Token:      testTagToken,
		ResourceID: testTagResourceID,
		Org:        testTagOrg,
		Client:     testTagClient,
	}
)

func TestNewTagger(t *testing.T) {
	actual, err := NewTagger(testTagEndpoint, testTagToken, testTagResourceID, testTagOrg, testTagEncryptToken)
	if err != nil {
		t.Errorf("Expected nil error for new tagger, got %s", err)
	}

	actual.Client = testTagClient

	if !reflect.DeepEqual(testTagger, actual) {
		t.Errorf("Expected NewDestroyer to return %+v, got %+v", testTagger, actual)
	}
}

func TestTag(t *testing.T) {
	tags := map[string]string{
		"foo": "bar",
		"baz": "biz",
	}

	tagger, err := NewTagger(testTagEndpoint, testTagToken, testTagResourceID, testTagOrg, testTagEncryptToken)
	if err != nil {
		t.Errorf("Expected nil error for new tagger, got %s", err)
	}

	successClient := NewMockClient([]byte("ok"), 200)
	successClient.Method = http.MethodPut
	successClient.HeaderMap = map[string]string{
		"X-Forwarded-User": "reaper",
		"X-Auth-Token":     testTagToken,
		"Content-Type":     "application/json",
	}

	tagger.Client = successClient
	err = tagger.Tag(tags)
	if err != nil {
		t.Error("Expected successful tag, got", err)
	}

	errorClient := NewMockClient([]byte("fail"), http.StatusInternalServerError)
	tagger.Client = errorClient
	err = tagger.Tag(tags)
	if err == nil {
		t.Error("Expected 500 error from http client to cause an error, got success")
	}
}
