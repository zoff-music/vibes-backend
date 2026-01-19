package spotify

import (
	"net/url"
	"testing"
)

func TestGetOAuthURL(t *testing.T) {
	clientID := "test-client-id"
	clientSecret := "test-client-secret"
	redirectURI := "http://localhost:8080/callback"
	c := &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}

	state := "test-state"
	authURL := c.GetOAuthURL(state)

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse generated URL: %v", err)
	}

	q := u.Query()
	if got := q.Get("client_id"); got != clientID {
		t.Errorf("Expected client_id %q, got %q", clientID, got)
	}
	if got := q.Get("redirect_uri"); got != redirectURI {
		t.Errorf("Expected redirect_uri %q, got %q", redirectURI, got)
	}
	if got := q.Get("state"); got != state {
		t.Errorf("Expected state %q, got %q", state, got)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Errorf("Expected response_type 'code', got %q", got)
	}

	expectedScope := "user-read-playback-state user-modify-playback-state"
	if got := q.Get("scope"); got != expectedScope {
		t.Errorf("Expected scope %q, got %q", expectedScope, got)
	}
}
