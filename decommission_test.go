package main

import (
	"net/http"
	"reflect"
	"testing"
)

var (
	testDecomEndpoint     = "http://127.0.0.1/v1/servers"
	testDecomToken        = "xypdq"
	testDecomResourceID   = "i-abc123"
	testDecomOrg          = "complaintdept"
	testDecomEncryptToken = false
	testDecomClient       = NewMockClient([]byte("ok"), 200)
	testDecommissioner    = Decommissioner{
		Endpoint:   testDecomEndpoint,
		Token:      testDecomToken,
		ResourceID: testDecomResourceID,
		Org:        testDecomOrg,
		Client:     testDecomClient,
	}
)

func TestNewDecommissioner(t *testing.T) {
	actual, err := NewDecommissioner(testDecomEndpoint, testDecomToken, testDecomResourceID, testDecomOrg, testDecomEncryptToken)
	if err != nil {
		t.Errorf("Expected nil error for new decommissioner, got %s", err)
	}

	actual.Client = testDecomClient

	if !reflect.DeepEqual(testDecommissioner, actual) {
		t.Errorf("Expected NewDecommissioner to return %+v, got %+v", testDecommissioner, actual)
	}
}

func TestSetStatus(t *testing.T) {
	decom, err := NewDecommissioner(testDecomEndpoint, testDecomToken, testDecomResourceID, testDecomOrg, testDecomEncryptToken)
	if err != nil {
		t.Errorf("Expected nil error for new decommissioner, got %s", err)
	}

	successClient := NewMockClient([]byte("ok"), 200)
	successClient.Method = http.MethodPut
	successClient.HeaderMap = map[string]string{
		"X-Forwarded-User": "reaper",
		"Auth-token":       testDecomToken,
		"Content-Type":     "application/json",
	}

	decom.Client = successClient
	err = decom.SetStatus()
	if err != nil {
		t.Error("Expected successful decom, got", err)
	}

	errorClient := NewMockClient([]byte("fail"), 500)
	decom.Client = errorClient
	err = decom.SetStatus()
	if err == nil {
		t.Error("Expected 500 error from http client to cause an error, got success")
	}
}
