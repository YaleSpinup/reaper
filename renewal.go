package main

import (
	"encoding/base64"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func generateRenewalToken(renewedAt, secret string) (string, error) {
	str, err := json.Marshal(RenewalSecret{
		RenewedAt: renewedAt,
		Secret:    secret,
	})
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
