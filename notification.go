package main

import (
	log "github.com/sirupsen/logrus"
)

// Notification is the data object for notifying
type Notfication struct {
	Endpoint string
	Token    string
	Params   map[string]string
}

// NewNotification creates a new notification object with the given params
func NewNotification(endpoint, token string, params map[string]string) *Notfication {
	return &Notfication{
		Endpoint: endpoint,
		Token:    token,
		Params:   params,
	}
}

// Notify sends the notification
func (n *Notfication) Notify() error {
	log.Debugf("Notifying with %+v", *n)
	return nil
}
