package helper

import (
	"fmt"
	"net/http"
	"time"
)

// SetOAuthStateCookie sets a secure cookie with the OAuth state
func SetOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthAuthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	})
}

// GetOAuthStateCookie retrieves the OAuth state from the cookie
func GetOAuthStateCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(oauthAuthStateCookie)
	if err != nil {
		return "", fmt.Errorf("error getting OAuth state cookie: %w", err)
	}
	return cookie.Value, nil
}

// ClearOAuthStateCookie clears the OAuth state cookie
func ClearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthAuthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

const oauthAuthStateCookie = "oauth_auth_state"
