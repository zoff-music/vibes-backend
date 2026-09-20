package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

func SignAdminAuthPayload(payload vibe.AdminAuthPayload, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("error signing admin payload: secret is required")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling admin payload: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(raw)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func ParseAdminAuthPayload(value string, secret string) (*vibe.AdminAuthPayload, error) {
	unsigned, err := unsignAdminPayload(value, secret)
	if err != nil {
		return nil, fmt.Errorf("error verifying admin payload signature: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(unsigned)
	if err != nil {
		return nil, fmt.Errorf("error decoding admin payload: %w", err)
	}

	var payload vibe.AdminAuthPayload
	err = json.Unmarshal(decoded, &payload)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling admin payload: %w", err)
	}

	if payload.UserID == "" ||
		payload.AdminID == "" ||
		payload.SessionVersion < 1 ||
		payload.IssuedAt < 1 {
		return nil, fmt.Errorf("error invalid admin payload")
	}

	return &payload, nil
}

func SignCastToken(secret string, payload vibe.CastTokenPayload) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("error cast token secret is required")
	}

	payload.V = 1
	payload.Typ = "cast"
	if payload.RoomID == "" || payload.UserID == "" {
		return "", fmt.Errorf("error cast token payload missing roomId/userId")
	}
	if payload.Iat == 0 {
		payload.Iat = time.Now().Unix()
	}
	if payload.Exp == 0 {
		return "", fmt.Errorf("error cast token payload missing exp")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling cast token payload: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(raw)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadB64 + "." + sigB64, nil
}

func VerifyCastToken(secret string, token string, now time.Time) (*vibe.CastTokenPayload, error) {
	if secret == "" {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error missing cast token secret")}
	}

	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error malformed cast token")}
	}
	payloadB64 := parts[0]
	sigB64 := parts[1]

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error decoding cast token signature: %w", err)}
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error invalid cast token signature")}
	}

	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error decoding cast token payload: %w", err)}
	}

	var out vibe.CastTokenPayload
	err = json.Unmarshal(raw, &out)
	if err != nil {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error unmarshaling cast token payload: %w", err)}
	}

	if out.V != 1 || out.Typ != "cast" || out.RoomID == "" || out.UserID == "" {
		return nil, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error invalid cast token payload")}
	}

	if out.Exp <= now.Unix() {
		return nil, internalerror.ErrCastTokenExpired{Err: fmt.Errorf("error expired cast token")}
	}

	return &out, nil
}

func unsignAdminPayload(value string, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("error verifying admin payload: secret is required")
	}

	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("error invalid admin payload signature")
	}

	payload := parts[0]
	signature := parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", fmt.Errorf("error invalid admin payload signature")
	}

	return payload, nil
}

const AdminAuthCookieName string = "admin_session"
