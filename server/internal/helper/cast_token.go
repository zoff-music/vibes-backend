package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type CastTokenPayload struct {
	V      int    `json:"v"`
	Typ    string `json:"typ"`
	RoomID string `json:"roomId"`
	UserID string `json:"userId"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}

var (
	ErrCastTokenInvalid = errors.New("cast token invalid")
	ErrCastTokenExpired = errors.New("cast token expired")
)

func SignCastToken(secret string, payload CastTokenPayload) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("cast token secret is required")
	}

	payload.V = 1
	payload.Typ = "cast"
	if payload.RoomID == "" || payload.UserID == "" {
		return "", fmt.Errorf("cast token payload missing roomId/userId")
	}
	if payload.Iat == 0 {
		payload.Iat = time.Now().Unix()
	}
	if payload.Exp == 0 {
		return "", fmt.Errorf("cast token payload missing exp")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal cast token payload: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(raw)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadB64 + "." + sigB64, nil
}

func VerifyCastToken(secret string, token string, now time.Time) (CastTokenPayload, error) {
	var out CastTokenPayload
	if secret == "" {
		return out, ErrCastTokenInvalid
	}

	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return out, ErrCastTokenInvalid
	}
	payloadB64 := parts[0]
	sigB64 := parts[1]

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return out, ErrCastTokenInvalid
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return out, ErrCastTokenInvalid
	}

	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return out, ErrCastTokenInvalid
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, ErrCastTokenInvalid
	}

	if out.V != 1 || out.Typ != "cast" || out.RoomID == "" || out.UserID == "" {
		return CastTokenPayload{}, ErrCastTokenInvalid
	}

	if out.Exp <= now.Unix() {
		return out, ErrCastTokenExpired
	}

	return out, nil
}
