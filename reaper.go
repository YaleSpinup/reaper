package main

import "net/http"

// HTTPClient abstracts the http client into an interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
