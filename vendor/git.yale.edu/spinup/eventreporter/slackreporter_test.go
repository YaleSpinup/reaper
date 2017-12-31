package eventreporter

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"
)

type MockClient struct {
	Response []byte
	Code     int
}

func NewMockClient(b []byte, c int) MockClient {
	return MockClient{
		Response: b,
		Code:     c,
	}
}

func (m MockClient) Do(req *http.Request) (*http.Response, error) {
	var res http.Response
	res.Body = ioutil.NopCloser(bytes.NewReader(m.Response))
	res.StatusCode = m.Code
	time.Sleep(100 * time.Millisecond)
	return &res, nil
}

var testSlackOptions = map[string]string{
	"endpoint": "https://slack.local/services/12345/6789",
	"channel":  "#thebikeshed",
	"username": "tester",
	"icon":     ":poop:",
}

var testSlackReporter = SlackReporter{
	Endpoint: "https://slack.local/services/12345/6789",
	Channel:  "#thebikeshed",
	Username: "tester",
	Icon:     ":poop:",
	Client:   NewMockClient([]byte("ok"), 200),
}

func TestNewSlackReporter(t *testing.T) {
	actual, err := NewSlackReporter(testSlackOptions)
	if err != nil {
		t.Error("Failed to generate slack reporter")
	}
	actual.Client = NewMockClient([]byte("ok"), 200)

	if !reflect.DeepEqual(&testSlackReporter, actual) {
		t.Errorf("Expected NewSlackReporter to return %+v, got %+v", testSlackReporter, actual)
	}
}

func TestSlackReport(t *testing.T) {
	actual, err := NewSlackReporter(testSlackOptions)
	if err != nil {
		t.Error("Failed to generate slack reporter")
	}
	actual.Client = NewMockClient([]byte("ok"), 200)

	err = actual.Report(Event{
		Message: "Awesome stuff is happening!",
		Level:   INFO,
	})

	if err != nil {
		t.Error("Failed to report event,", err)
	}

	actual.Client = NewMockClient([]byte("error"), 500)

	err = actual.Report(Event{
		Message: "Oh noes!",
		Level:   INFO,
	})

	if err == nil {
		t.Error("Expected error when non-success http response, but got nil")
	}
}

func BenchmarkSlackReport(b *testing.B) {
	actual, err := NewSlackReporter(testSlackOptions)
	if err != nil {
		b.Error("Failed to generate slack reporter")
	}
	actual.Client = NewMockClient([]byte("ok"), 200)

	// run the reporter function b.N times
	for n := 0; n < b.N; n++ {
		err = actual.Report(Event{
			Message: "Awesome stuff is happening!",
			Level:   INFO,
		})

		if err != nil {
			b.Error("Failed to report event,", err)
		}
	}
}
