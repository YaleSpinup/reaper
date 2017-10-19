package common

import (
	"bytes"
	"reflect"
	"testing"
)

var testConfig = []byte(
	`{ 
		"providers": {
			"provider1": {
				"type": "http",
				"endpoint": "http://127.0.0.1:1234/v1",
				"provider": "ec2",
				"account": "nonprod",
				"token": "test",
				"timeout": "60s"
			},
			"provider2": {
				"type": "file",
				"path": "/tmp/foo",
				"provider": "file",
				"account": "sandbox"
			}
		},
		"engine": {
			"endpoint": "http://127.0.0.1:9200/foobar",
			"timeout": "60s"
		},
		"filter": {
			"tagKeyValues": {
				"foo": ["bar", "baz"],
				"hide:andseek": []
			}
		},
		"dryRun": false
	}`)

func TestReadConfig(t *testing.T) {
	expectedConfig := Config{
		Providers: map[string]map[string]string{
			"provider1": map[string]string{
				"type":     "http",
				"endpoint": "http://127.0.0.1:1234/v1",
				"provider": "ec2",
				"account":  "nonprod",
				"token":    "test",
				"timeout":  "60s",
			},
			"provider2": map[string]string{
				"type":     "file",
				"path":     "/tmp/foo",
				"provider": "file",
				"account":  "sandbox",
			},
		},
		Engine: map[string]string{
			"endpoint": "http://127.0.0.1:9200/foobar",
			"timeout":  "60s",
		},
		Filter: Filter{
			TagKeyValues: map[string][]string{
				"foo": []string{
					"bar",
					"baz",
				},
				"hide:andseek": []string{},
			},
		},
	}

	actualConfig, err := ReadConfig(bytes.NewReader(testConfig))
	if err != nil {
		t.Error("Failed to read config", err)
	}

	if !reflect.DeepEqual(actualConfig, expectedConfig) {
		t.Errorf("expected: %+v", expectedConfig)
		t.Errorf("actual:   %+v", actualConfig)
	}
}
