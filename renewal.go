package main

import (
	"encoding/base64"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// RenewalSecret is the object used to generate the renewal token
type RenewalSecret struct {
	RenewedAt string `json:"renewed_at"`
	Secret    string `json:"secret"`
}

// GenerateRenewalToken creates a base64 encoded renewal token
func (r *RenewalSecret) GenerateRenewalToken() (string, error) {
	str, err := json.Marshal(r)
	if err != nil {
		return "", err
	}

	log.Debugf("Marshalled secret JSON string %s", str)

	token, err := bcrypt.GenerateFromPassword(str, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	log.Debugln("Secret hash:", string(token))

	return base64.StdEncoding.EncodeToString(token), nil
}

// ValidateRenewalToken validates a base64 encoded renewal token
func (r *RenewalSecret) ValidateRenewalToken(token string) error {
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		log.Error("Failed to decode base64", err)
		return err
	}

	str, err := json.Marshal(r)
	if err != nil {
		log.Error("Failed to marshall JSON", err)
		return err
	}

	log.Debugf("Comparing decodedToken: %s with renewalSecret %s", decodedToken, string(str))

	return bcrypt.CompareHashAndPassword(decodedToken, str)
}
