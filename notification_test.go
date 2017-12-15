package main

import "testing"
import "reflect"

func TestNewNotifier(t *testing.T) {
	expected := Notifier{
		Endpoint: "http://127.0.0.1:8888/v1/notify",
		Token:    "12345",
	}

	actual := NewNotifier("http://127.0.0.1:8888/v1/notify", "12345")

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %+v, got %+v", expected, actual)
	}
}
