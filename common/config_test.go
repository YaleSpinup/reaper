package common

import (
	"bytes"
	"reflect"
	"testing"
)

var testConfig = []byte(
	`{
		"listen": "127.0.0.1:8080",
		"searchEngine": {
		  "endpoint": "http://127.0.0.1:9200"
		},
		"userDatasource": {
			"type": "rest",
			"endpoint": "http://127.0.0.1:8888/v1/users",
			"token": "12345"
		},
		"filter": {
		  "yale:subsidized": "true"
		},
		"email": {
			"mailserver": "nofilter.thomas.hooker",
			"from": "Nummy Nummy <nummy@stubborn.beauty>",
			"username": "CounterWeight",
			"password": "HeadWay"
		},
		"notify": {
		  "age": [
			"1d",
			"20d",
			"5d",
			"25d"
		  ]
		},
		"decommission": {
		  "age": "30d",
		  "endpoint": "http://127.0.0.1:8888/v1/decom",
		  "token": "12345"
		},
		"destroy": {
		  "age": "44d",
		  "endpoint": "http://127.0.0.1:8888/v1/destroy",
		  "token": "12345"
		},
		"tagging": {
		  "endpoint": "http://127.0.0.1:8888/v1/servers",
		  "token": "12345"
		},
		"interval": "15s",
		"logLevel": "debug",
		"baseUrl": "http://127.0.0.1:8080/v1/reaper",
		"encryptionSecret": "54321"
	  }`)

func TestReadConfig(t *testing.T) {
	expectedConfig := Config{
		BaseURL: "http://127.0.0.1:8080/v1/reaper",
		Decommission: Decommissioner{
			Age:      "30d",
			Endpoint: "http://127.0.0.1:8888/v1/decom",
			Token:    "12345",
		},
		Destroy: Destroyer{
			Age:      "44d",
			Endpoint: "http://127.0.0.1:8888/v1/destroy",
			Token:    "12345",
		},
		Email: Emailer{
			Mailserver: "nofilter.thomas.hooker",
			From:       "Nummy Nummy <nummy@stubborn.beauty>",
			Username:   "CounterWeight",
			Password:   "HeadWay",
		},
		Filter:   map[string]string{"yale:subsidized": "true"},
		Interval: "15s",
		Listen:   "127.0.0.1:8080",
		LogLevel: "debug",
		Notify: Notifier{
			Age: []string{"1d", "20d", "5d", "25d"},
		},
		SearchEngine: map[string]string{"endpoint": "http://127.0.0.1:9200"},
		UserDatasource: map[string]string{
			"type":     "rest",
			"endpoint": "http://127.0.0.1:8888/v1/users",
			"token":    "12345",
		},
		Tagging: Tagging{
			Endpoint: "http://127.0.0.1:8888/v1/servers",
			Token:    "12345",
		},
		EncryptionSecret: "54321",
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
