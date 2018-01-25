package eventreporter

import (
	"net/smtp"
	"reflect"
	"testing"
)

var testEmailOptionsPlain = map[string]string{
	"mailserver": "foo.bar.com:25",
	"username":   "moar@theclouds.com",
	"password":   "1$B3ttar",
	"to":         "azr@cloud.com;aws@thecloud.com;goog@thecloud.com",
	"from":       "on@prem.biz",
}

var testEmailReporter = EmailReporter{
	Address: "foo.bar.com:25",
	Auth:    smtp.PlainAuth("", "moar@theclouds.com", "1$B3ttar", "foo.bar.com"),
	From:    "on@prem.biz",
	To:      []string{"azr@cloud.com", "aws@thecloud.com", "goog@thecloud.com"},
}

func TestNewEmailReporter(t *testing.T) {
	actual, err := NewEmailReporter(testEmailOptionsPlain)
	if err != nil {
		t.Error("Failed to generate email reporter")
	}

	if !reflect.DeepEqual(&testEmailReporter, actual) {
		t.Errorf("Expected NewEmailReporter to return %+v, got %+v", testEmailReporter, actual)
	}
	t.Log("Got expected test email reporter:", actual)

	_, err = NewEmailReporter(map[string]string{
		"mailserver": "foo.bar.com",
		"username":   "moar@theclouds.com",
		"password":   "1$B3ttar",
		"to":         "azr@cloud.com;aws@thecloud.com;goog@thecloud.com",
		"from":       "on@prem.biz",
	})
	if err == nil {
		t.Error("Expected malformed 'mailserver' to throw an error")
	}
	t.Log("Got expected error:", err)

	_, err = NewEmailReporter(map[string]string{
		"mailserver": "foo.bar.com:25",
		"username":   "moar@theclouds.com",
		"password":   "1$B3ttar",
		"to":         "azr@cloud.com;aws@thecloud.com;goog@thecloud.com",
		"from":       "300moar",
	})
	if err == nil {
		t.Error("Expected malformed 'from' to throw an error")
	}
	t.Log("Got expected error:", err)

	_, err = NewEmailReporter(map[string]string{
		"mailserver": "foo.bar.com:25",
		"username":   "moar@theclouds.com",
		"password":   "1$B3ttar",
		"from":       "300moar",
	})
	if err == nil {
		t.Error("Expected missing 'to' to throw an error")
	}
	t.Log("Got expected error:", err)
}
