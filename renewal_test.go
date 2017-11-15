package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateRenewalToken(t *testing.T) {
	renewalSecret := &RenewalSecret{RenewedAt: "2013/06/19 19:14:05", Secret: "54321"}
	encodedToken, err := renewalSecret.GenerateRenewalToken()
	if err != nil {
		t.Error("Failed to generate renewal token:", err)
	}

	actualToken, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		t.Error("Failed to decode base64 encoded string", err)
	}

	str, err := json.Marshal(renewalSecret)
	if err != nil {
		t.Error("Failed to marshall JSON", err)
	}

	err = bcrypt.CompareHashAndPassword(actualToken, str)
	if err != nil {
		t.Errorf("Failed to compare generated hash %s and string token %s. %s", actualToken, string(str), err)
	}
}

func TestValidateRenewalToken(t *testing.T) {
	// Generate the test passed "password"
	renewalSecret1 := &RenewalSecret{RenewedAt: "2013/06/19 19:14:05", Secret: "54321"}
	str, err := json.Marshal(renewalSecret1)
	if err != nil {
		t.Error("Failed to marshall JSON", err)
	}

	ct, err := bcrypt.GenerateFromPassword(str, bcrypt.DefaultCost)
	if err != nil {
		t.Error("Failed to generate from password", err)
	}

	password := base64.StdEncoding.EncodeToString(ct)

	// Generate the renewal secret to compare
	renewalSecret2 := &RenewalSecret{RenewedAt: "2013/06/19 19:14:05", Secret: "54321"}
	t.Logf("Validating token %s against renewal secert %+v", password, renewalSecret2)
	if err := renewalSecret2.ValidateRenewalToken(password); err != nil {
		t.Errorf("Expected valid token to validate. Token: %s Secret: %s. %s", password, renewalSecret2, err.Error())
	}
}
