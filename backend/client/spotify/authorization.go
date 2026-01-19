package spotify

import "net/url"

// GetOAuthURL returns the URL to redirect the user to for Spotify authentication
func (c *Client) GetOAuthURL(state string) string {
	u, _ := url.Parse("https://accounts.spotify.com/authorize")
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", c.clientID)
	q.Set("scope", "user-read-playback-state user-modify-playback-state")
	q.Set("redirect_uri", c.redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()

	return u.String()
}
