package eventreporter

import "net/http"

type Level uint32

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

type Event struct {
	Message string
	Level   Level
}

type Reporter interface {
	Report(*Event) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
