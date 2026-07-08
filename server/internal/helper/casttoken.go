package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
)

type CastTokenPayload struct {
	V      int    `json:"v"`
	Typ    string `json:"typ"`
	RoomID string `json:"roomId"`
	UserID string `json:"userId"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}

func SignCastToken(secret string, payload CastTokenPayload) (string, error) {
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

func VerifyCastToken(secret string, token string, now time.Time) (CastTokenPayload, error) {
	if secret == "" {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error missing cast token secret")}
	}

	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error malformed cast token")}
	}
	payloadB64 := parts[0]
	sigB64 := parts[1]

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error decoding cast token signature: %w", err)}
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error invalid cast token signature")}
	}

	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error decoding cast token payload: %w", err)}
	}

	var out CastTokenPayload
	err = json.Unmarshal(raw, &out)
	if err != nil {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error unmarshaling cast token payload: %w", err)}
	}

	if out.V != 1 || out.Typ != "cast" || out.RoomID == "" || out.UserID == "" {
		return CastTokenPayload{}, internalerror.ErrCastTokenInvalid{Err: fmt.Errorf("error invalid cast token payload")}
	}

	if out.Exp <= now.Unix() {
		return CastTokenPayload{}, internalerror.ErrCastTokenExpired{Err: fmt.Errorf("error expired cast token")}
	}

	return out, nil
}
