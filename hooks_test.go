package main

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/reaper/common"
)

var (
	testHookEndpoint = "http://127.0.0.1/v1/hook"
	testHookToken    = "xypdq"
	testHookID       = "i-abc123"
	testHookClient   = NewMockClient([]byte("ok"), 200)
)

func TestNewWebhook(t *testing.T) {
	testWebhook := Webhook{
		Endpoint: "http://127.0.0.1/v1/hook",
		Token:    "xypdq",
		Method:   "POST",
		Actions:  []string{"party"},
		Client:   testHookClient,
	}

	actual, err := NewWebhook(common.Webhook{
		Endpoint: "http://127.0.0.1/v1/hook",
		Token:    "xypdq",
		Method:   "POST",
		Actions:  []string{"party"},
	})
	actual.Client = testHookClient

	if err != nil {
		t.Errorf("got error %s, expected nil", err)
	}

	if !reflect.DeepEqual(testWebhook, actual) {
		t.Errorf("Expected NewWebhook to return %+v, got %+v", testWebhook, actual)
	}
}
