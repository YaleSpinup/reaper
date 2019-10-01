package main

import "testing"

var testEamilParams = map[string]string{
	"first":         "bob",
	"netid":         "abc123",
	"fqdn":          "foo.bar.yale.edu",
	"link":          "https://127.0.0.1:8888/v1/renew",
	"expire_on":     "2018/01/14 15:32:24",
	"spinupURL":     "http://127.0.0.1:8888/spinup",
	"spinupSiteURL": "http://127.0.0.1:8888/spinup",
}

func TestParseWarningTemplate(t *testing.T) {
	out, err := ParseWarningTemplate(testEamilParams)
	if err != nil {
		t.Error("Failed to parse warning temapte", err)
	}

	t.Logf("Got parsed warning template: %s\n", out)
}

func TestParseRenewalTemplate(t *testing.T) {
	out, err := ParseRenewalTemplate(testEamilParams)
	if err != nil {
		t.Error("Failed to parse warning temapte", err)
	}

	t.Logf("Got parsed warning template: %s\n", out)
}

func TestParseDecomTemplate(t *testing.T) {
	out, err := ParseDecomTemplate(testEamilParams)
	if err != nil {
		t.Error("Failed to parse warning temapte", err)
	}

	t.Logf("Got parsed warning template: %s\n", out)
}
