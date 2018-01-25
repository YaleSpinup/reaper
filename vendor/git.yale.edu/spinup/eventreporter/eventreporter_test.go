package eventreporter_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"

	log "github.com/sirupsen/logrus"

	report "git.yale.edu/spinup/eventreporter"
)

var eventReportersList = map[string]map[string]string{
	"slack": {
		"endpoint": "http://127.0.0.1:8888/v1/slack",
		"channel":  "#devchannel",
		"icon":     ":see_no_evil:",
		"username": "slacker",
	},
}

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

func init() {
	log.SetLevel(log.WarnLevel)
}

func TestEventReporter(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	defer leaktest.CheckContext(ctx, t)()

	reporters, err := configureEventReporters()
	if err != nil {
		t.Errorf("Couldn't initialize event reporters: %s", err.Error())
	}

	reportEvent(reporters, "Some message being reported", report.INFO)
}

func TestConcurrentEventReporter(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	defer leaktest.CheckContext(ctx, t)()

	reporters, err := configureEventReporters()
	if err != nil {
		t.Errorf("Couldn't initialize event reporters: %s", err.Error())
	}

	// run the reporter function 10 times
	for i := 0; i < 10; i++ {
		var wg sync.WaitGroup
		wg.Add(4)
		go func() {
			defer wg.Done()
			reportEvent(reporters, "Some superfluous event", report.DEBUG)
		}()

		go func() {
			defer wg.Done()
			reportEvent(reporters, "Some interesting event", report.INFO)
		}()

		go func() {
			defer wg.Done()
			reportEvent(reporters, "Some curious event", report.WARN)
		}()

		go func() {
			defer wg.Done()
			reportEvent(reporters, "Some terrifying event", report.ERROR)
		}()

		wg.Wait()
	}
}

func BenchmarkEventReporter(b *testing.B) {
	reporters, err := configureEventReporters()
	if err != nil {
		b.Errorf("Couldn't initialize event reporters: %s", err.Error())
	}

	// run the reporter function b.N times
	for n := 0; n < b.N; n++ {
		reportEvent(reporters, "Some interesting event", report.INFO)
	}
}

func BenchmarkEventReporterConcurrent(b *testing.B) {
	reporters, err := configureEventReporters()
	if err != nil {
		b.Errorf("Couldn't initialize event reporters: %s", err.Error())
	}

	// run the reporter function b.N times
	for n := 0; n < b.N; n++ {
		for i := 0; i < 10; i++ {
			var wg sync.WaitGroup
			wg.Add(4)
			go func() {
				defer wg.Done()
				reportEvent(reporters, "Some superfluous event", report.DEBUG)
			}()

			go func() {
				defer wg.Done()
				reportEvent(reporters, "Some interesting event", report.INFO)
			}()

			go func() {
				defer wg.Done()
				reportEvent(reporters, "Some curious event", report.WARN)
			}()

			go func() {
				defer wg.Done()
				reportEvent(reporters, "Some terrifying event", report.ERROR)
			}()

			wg.Wait()
		}
	}
}

func configureEventReporters() ([]report.Reporter, error) {
	var reporters []report.Reporter
	for name, config := range eventReportersList {
		switch name {
		case "slack":
			reporter, err := report.NewSlackReporter(config)
			if err != nil {
				return nil, err
			}
			reporter.Client = NewMockClient([]byte("ok"), 200)
			reporters = append(reporters, reporter)
		default:
			msg := fmt.Sprintf("Unknown event reporter name, %s", name)
			return reporters, errors.New(msg)
		}
	}

	return reporters, nil
}

func reportEvent(reporters []report.Reporter, msg string, level report.Level) error {
	e := report.Event{
		Message: msg,
		Level:   level,
	}

	for _, r := range reporters {
		err := r.Report(e)
		if err != nil {
			return err
		}
	}

	return nil
}
