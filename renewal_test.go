package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateRenewalToken(t *testing.T) {
	encodedToken, err := generateRenewalToken("2013/06/19 19:14:05", "54321")
	if err != nil {
		t.Error("Failed to generate renewal token:", err)
	}
	actualToken, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		t.Error("Failed to decode base64 encoded string", err)
	}

	str, err := json.Marshal(RenewalSecret{RenewedAt: "2013/06/19 19:14:05", Secret: "54321"})
	if err != nil {
		t.Error("Failed to marshall JSON", err)
	}

	err = bcrypt.CompareHashAndPassword(actualToken, str)
	if err != nil {
		t.Errorf("Failed to compare generated hash %s and string token %s. %s", actualToken, string(str), err)
	}
}
