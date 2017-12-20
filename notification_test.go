package main

import "testing"
import "reflect"

func TestNewNotifier(t *testing.T) {
	client := NewMockClient([]byte("ok"), 200)
	expected := Notifier{
		Endpoint: "http://127.0.0.1:8888/v1/notify",
		Token:    "12345",
		Client:   client,
	}

	actual := NewNotifier("http://127.0.0.1:8888/v1/notify", "12345")
	actual.Client = client

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %+v, got %+v", expected, actual)
	}
}
