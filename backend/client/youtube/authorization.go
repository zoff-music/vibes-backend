package youtube

import "net/url"

// GetOAuthURL returns the URL to redirect the user to for YouTube (Google) authentication
func (c *Client) GetOAuthURL(state string) string {
	u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", c.clientID)
	q.Set("scope", "https://www.googleapis.com/auth/youtube.readonly")
	q.Set("redirect_uri", c.redirectURI)
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	u.RawQuery = q.Encode()

	return u.String()
}
