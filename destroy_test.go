package main

import (
	"net/http"
	"reflect"
	"testing"
)

var (
	testDestroyEndpoint   = "http://127.0.0.1/v1/servers"
	testDestroyToken      = "xypdq"
	testDestroyResourceID = "i-abc123"
	testDestroyOrg        = "complaintdept"
	testDestroyClient     = NewMockClient([]byte("ok"), 200)
	testDestroyer         = Destroyer{
		Endpoint:   testDestroyEndpoint,
		Token:      testDestroyToken,
		ResourceID: testDestroyResourceID,
		Org:        testDestroyOrg,
		Client:     testDestroyClient,
	}
)

func TestNewDestroyer(t *testing.T) {
	actual := NewDestroyer(testDestroyEndpoint, testDestroyToken, testDestroyResourceID, testDestroyOrg)
	actual.Client = testDestroyClient

	if !reflect.DeepEqual(testDestroyer, actual) {
		t.Errorf("Expected NewDestroyer to return %+v, got %+v", testDestroyer, actual)
	}
}

func TestDestroy(t *testing.T) {
	destroyerofworlds := NewDestroyer(testDestroyEndpoint, testDestroyToken, testDestroyResourceID, testDestroyOrg)

	successClient := NewMockClient([]byte("ok"), 200)
	successClient.Method = http.MethodDelete
	successClient.HeaderMap = map[string]string{
		"X-Forwarded-User": "reaper",
		"Auth-token":       testDestroyToken,
		"Content-Type":     "application/json",
	}

	destroyerofworlds.Client = successClient
	err := destroyerofworlds.Destroy()
	if err != nil {
		t.Error("Expected successful destruction, got", err)
	}

	errorClient := NewMockClient([]byte("fail"), 500)
	destroyerofworlds.Client = errorClient
	err = destroyerofworlds.Destroy()
	if err == nil {
		t.Error("Expected 500 error from http client to cause an error, got success")
	}
}
