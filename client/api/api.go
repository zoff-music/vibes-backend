// Package api contains an api api client.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zoff-music/cibes/monitoring/opentracing"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/zoff-music/cibes/client"
	"github.com/zoff-music/cibes/config"
)

// Client holds the api client.
type Client struct {
	Endpoint   string
	HTTPClient client.HTTPClient
	// Access
	AccessEndpoint string
	ClientID       string
	ClientSecret   string
	accessToken    accessToken
}

// Init initializes a new api client.
func (c *Client) Init(config *config.Config) error {
	timeout := 5 * time.Second
	c.Endpoint = config.ExampleAPIEndpoint
	c.HTTPClient = client.NewHTTPClient(client.Parameters{Timeout: &timeout})

	c.ClientID = config.ExampleAPIClientID
	c.ClientSecret = config.ExampleAPIClientSecret
	c.AccessEndpoint = config.ExampleAPIAccessEndpoint
	// Commented out so the code runs
	//err := c.getAccessToken(context.Background())
	//if err != nil {
	//	return fmt.Errorf("error getting access token: %w", err)
	//}
	return nil
}

func (c *Client) getAccessToken(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "getAccessToken")
	defer span.Finish()

	c.accessToken.mux.Lock()
	defer c.accessToken.mux.Unlock()
	// If the token doesn't expire in the next 10 seconds, move on
	if time.Now().Add(10 * time.Second).Before(c.accessToken.token.expiresAt) {
		return nil
	}

	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.AccessEndpoint,
		Headers: map[string]string{
			"client_id":     c.ClientID,
			"client_secret": c.ClientSecret,
		},
	}

	respBody, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return fmt.Errorf("error making request to %s for token: %w", c.AccessEndpoint, err)
	}

	var rawToken rawAccessToken
	err = json.Unmarshal(respBody, &rawToken)
	if err != nil {
		return fmt.Errorf("error unmarshalling rawAccessToken: %w", err)
	}

	expiresIn, err := strconv.Atoi(rawToken.ExpiresIn)
	if err != nil {
		return fmt.Errorf("error getting expiring date for raw token: %w", err)
	}

	c.accessToken.token = token{
		value:     rawToken.Token,
		expiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	}

	return nil
}

type accessToken struct {
	mux   sync.Mutex
	token token
}

type token struct {
	value     string
	expiresAt time.Time
}

type rawAccessToken struct {
	Token     string `json:"access_token"`
	ExpiresIn string `json:"expires_in"`
}
