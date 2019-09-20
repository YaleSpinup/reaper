package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/reaper/common"
)

var (
	testHookToken  = "xypdq"
	testHookClient = NewMockClient([]byte("ok"), 200)
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

func TestSendWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := Event{}

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodDelete, http.MethodOptions:
			q := r.URL.Query()
			event.ID = q.Get("id")
			event.Action = q.Get("action")
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			err := json.NewDecoder(r.Body).Decode(&event)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("unable to decode JSON request: " + err.Error()))
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("unexpected method: " + r.Method))
			return
		}

		t.Logf("received %s event from webhook: %+v", r.Method, event)

		if event.ID != "i-123456789" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("bad id: " + event.ID + ", expected 'i-123456789'"))
			return
		}

	}))
	defer server.Close()

	wh := Webhook{
		Endpoint: server.URL,
		Token:    testHookToken,
		Method:   http.MethodPost,
		Actions:  []string{"party"},
		Client: &http.Client{
			Timeout: time.Second * 3,
		},
	}

	// other action should just not send the hook
	wh.Method = http.MethodPost
	if err := wh.Send(context.TODO(), &Event{
		Action: "dance",
		ID:     "i-123456789",
	}); err != nil {
		t.Errorf("expected error for bad method, got nil")
	}

	// post-like should expect a body
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch} {
		wh.Method = m
		if err := wh.Send(context.TODO(), &Event{
			Action: "party",
			ID:     "i-123456789",
		}); err != nil {
			t.Errorf("expected nil error for valid hook, got %s", err)
		}
	}

	// get-like should expect a url
	for _, m := range []string{http.MethodGet, http.MethodHead} {
		wh.Method = m
		if err := wh.Send(context.TODO(), &Event{
			Action: "party",
			ID:     "i-123456789",
		}); err != nil {
			t.Errorf("expected nil error for valid hook, got %s", err)
		}
	}

	// invalid methods should throw an error
	for _, m := range []string{http.MethodTrace, http.MethodConnect} {
		wh.Method = m
		if err := wh.Send(context.TODO(), &Event{
			Action: "party",
			ID:     "i-123456789",
		}); err == nil {
			t.Errorf("expected error for bad method, got nil")
		}
	}
}
