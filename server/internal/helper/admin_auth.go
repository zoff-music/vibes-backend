package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const AdminAuthCookieName string = "admin_session"

type AdminAuthPayload struct {
	UserID       string `json:"user_id"`
	PasswordHash string `json:"password_hash"`
	IssuedAt     int64  `json:"issued_at"`
}

func HashAdminPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func SignAdminAuthPayload(payload AdminAuthPayload, secret string) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling admin payload: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(raw)
	if secret == "" {
		return encoded, nil
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func ParseAdminAuthPayload(value string, secret string) (AdminAuthPayload, error) {
	var payload AdminAuthPayload

	unsigned, err := unsignAdminPayload(value, secret)
	if err != nil {
		return payload, err
	}

	decoded, err := base64.StdEncoding.DecodeString(unsigned)
	if err != nil {
		return payload, fmt.Errorf("error decoding admin payload: %w", err)
	}

	err = json.Unmarshal(decoded, &payload)
	if err != nil {
		return payload, fmt.Errorf("error unmarshaling admin payload: %w", err)
	}

	if payload.UserID == "" || payload.PasswordHash == "" {
		return payload, fmt.Errorf("invalid admin payload")
	}

	return payload, nil
}

func unsignAdminPayload(value string, secret string) (string, error) {
	if secret == "" {
		return value, nil
	}

	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid admin payload signature")
	}

	payload := parts[0]
	signature := parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", fmt.Errorf("invalid admin payload signature")
	}

	return payload, nil
}
