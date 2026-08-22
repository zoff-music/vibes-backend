package client

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestBytesSetsApplicationUserAgent(t *testing.T) {
	tests := []struct {
		name              string
		expectedUserAgent string
	}{
		{
			name:              "identifies the vibes backend",
			expectedUserAgent: applicationName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedUserAgent string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedUserAgent = r.Header.Get("User-Agent")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			httpClient := HTTPClient{Client: server.Client()}
			_, err := httpClient.RequestBytes(t.Context(), HTTPRequestData{
				Method: http.MethodGet,
				URL:    server.URL,
			})
			if err != nil {
				t.Fatalf("expected request to succeed: %v", err)
			}
			if receivedUserAgent != tt.expectedUserAgent {
				t.Fatalf("expected user agent %q, got %q", tt.expectedUserAgent, receivedUserAgent)
			}
		})
	}
}

func TestErrorCodeWrapperUsesApplicationNamespace(t *testing.T) {
	tests := []struct {
		name              string
		wrapper           ErrorCodeWrapper
		expectedNamespace string
	}{
		{
			name: "uses the vibes backend namespace by default",
			wrapper: ErrorCodeWrapper{
				Err: fmt.Errorf("error downstream request failed"),
				ResponseBody: ErrorCodeResponseBody{
					Error:   "downstream_error",
					Message: "request failed",
				},
			},
			expectedNamespace: applicationName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := tt.wrapper.GetResponseBody()
			if err != nil {
				t.Fatalf("expected error response to marshal: %v", err)
			}

			var response ErrorCodeResponseBody
			err = json.Unmarshal(body, &response)
			if err != nil {
				t.Fatalf("expected error response to unmarshal: %v", err)
			}
			if response.Namespace != tt.expectedNamespace {
				t.Fatalf("expected namespace %q, got %q", tt.expectedNamespace, response.Namespace)
			}
		})
	}
}
