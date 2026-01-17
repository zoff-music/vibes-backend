package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zoff-music/cibes/client"
	"github.com/zoff-music/cibes/example"
	"github.com/zoff-music/cibes/monitoring/opentracing"
	"net/http"
)

// GetExampleData gets example data from the example api.
func (c *Client) GetExampleData(ctx context.Context) (*example.Data, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetExampleData")
	defer span.Finish()

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/api/v1/example", c.Endpoint),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": c.accessToken.token.value,
		},
	}

	var exampleData example.Data
	respBody, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error making request to example api to get example data: %w", err)
	}
	err = json.Unmarshal(respBody, &exampleData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling example data: %w", err)
	}
	return &exampleData, nil
}
