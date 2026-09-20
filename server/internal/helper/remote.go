package helper

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

func GenerateRemoteToken() (string, error) {
	value := make([]byte, remoteTokenBytes)
	_, err := rand.Read(value)
	if err != nil {
		return "", fmt.Errorf("error generating remote token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(value)
	return token, nil
}

func GenerateRemoteCode() (string, error) {
	value := make([]byte, remoteCodeBytes)
	_, err := rand.Read(value)
	if err != nil {
		return "", fmt.Errorf("error generating remote code: %w", err)
	}

	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(value)

	normalizedCode := strings.ToUpper(code)
	return normalizedCode, nil
}

func HashRemoteCredential(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))

	hash := hex.EncodeToString(mac.Sum(nil))
	return hash
}

const remoteTokenBytes = 32

const remoteCodeBytes = 5
