package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

// MockClient is a mock HTTP client for testing
type MockClient struct {
	Response  []byte
	Code      int
	HeaderMap map[string]string
	Method    string
}

// NewMockClient creates a new mock client that will return the response b and the code c
func NewMockClient(b []byte, c int) MockClient {
	return MockClient{
		Response: b,
		Code:     c,
	}
}

// Do implements the HTTPClient interface for MockClient
func (m MockClient) Do(req *http.Request) (*http.Response, error) {
	var res http.Response

	if len(m.HeaderMap) > 0 {
		for k, value := range m.HeaderMap {
			fmt.Printf("Checking key: %s for values %+v against headers %+v", k, value, req.Header)

			if value != req.Header.Get(k) {
				res.Body = ioutil.NopCloser(bytes.NewReader([]byte("Bad header values")))
				res.StatusCode = http.StatusBadRequest
				return &res, nil
			}
		}
	}

	if m.Method != "" {
		if req.Method != m.Method {
			res.Body = ioutil.NopCloser(bytes.NewReader([]byte("Bad method")))
			res.StatusCode = http.StatusBadRequest
			return &res, nil
		}
	}

	res.Body = ioutil.NopCloser(bytes.NewReader(m.Response))
	res.StatusCode = m.Code
	return &res, nil
}
