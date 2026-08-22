package client

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingRoundTripper struct{}

func (f *failingRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("error connection refused")
}

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

func TestRedactBodyForLog(t *testing.T) {
	tests := []struct {
		name            string
		headers         map[string]string
		body            string
		expectedPresent []string
		expectedAbsent  []string
	}{
		{
			name:    "recursively redacts normalized JSON secret keys",
			headers: map[string]string{"content-type": "Application/Problem+JSON; charset=UTF-8"},
			body: `{
				"accessToken":"json-access-secret",
				"nested":{"code_verifier":"json-verifier-secret","safe":"visible"},
				"items":[{"controller-token":"json-controller-secret"}]
			}`,
			expectedPresent: []string{redactedValue, "visible"},
			expectedAbsent:  []string{"json-access-secret", "json-verifier-secret", "json-controller-secret"},
		},
		{
			name:    "redacts normalized form secret keys",
			headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded; charset=utf-8"},
			body:    "clientSecret=form-client-secret&pairing_token=form-pairing-secret&safe=visible",
			expectedPresent: []string{
				"safe=visible",
				"clientSecret=%5BREDACTED%5D",
				"pairing_token=%5BREDACTED%5D",
			},
			expectedAbsent: []string{"form-client-secret", "form-pairing-secret"},
		},
		{
			name:            "fails closed for malformed JSON",
			headers:         map[string]string{"Content-Type": "application/json"},
			body:            `{"accessToken":`,
			expectedPresent: []string{"[unparseable json body:"},
			expectedAbsent:  []string{"accessToken"},
		},
		{
			name:            "omits oversized redacted JSON",
			headers:         map[string]string{"Content-Type": "application/json"},
			body:            `{"safe":"` + strings.Repeat("a", maxLoggedValueBytes) + `"}`,
			expectedPresent: []string{"[redacted json body omitted:"},
			expectedAbsent:  []string{strings.Repeat("a", 100)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted := redactBodyForLog(tt.headers, []byte(tt.body))
			for _, expected := range tt.expectedPresent {
				if !strings.Contains(redacted, expected) {
					t.Fatalf("expected redacted body %q to contain %q", redacted, expected)
				}
			}
			for _, secret := range tt.expectedAbsent {
				if strings.Contains(redacted, secret) {
					t.Fatalf("expected redacted body %q not to contain %q", redacted, secret)
				}
			}
		})
	}
}

func TestRequestBytesRedactsHTTPFailureDetails(t *testing.T) {
	tests := []struct {
		name            string
		client          *http.Client
		url             string
		expectedAbsent  []string
		expectedMessage string
	}{
		{
			name: "redacts query secrets from transport errors",
			client: &http.Client{
				Transport: &failingRoundTripper{},
			},
			url: "https://transport-user:transport-password@example.com/search" +
				"?key=transport-query-secret&q=music#transport-fragment-secret",
			expectedAbsent: []string{
				"transport-user",
				"transport-password",
				"transport-query-secret",
				"transport-fragment-secret",
			},
			expectedMessage: "%5BREDACTED%5D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := HTTPClient{Client: tt.client}
			_, err := httpClient.RequestBytes(t.Context(), HTTPRequestData{
				Method: http.MethodGet,
				URL:    tt.url,
			})
			if err == nil {
				t.Fatal("expected request to fail")
			}
			message := err.Error()
			for _, secret := range tt.expectedAbsent {
				if strings.Contains(message, secret) {
					t.Fatalf("expected error %q not to contain %q", message, secret)
				}
			}
			if !strings.Contains(message, tt.expectedMessage) {
				t.Fatalf("expected error %q to contain %q", message, tt.expectedMessage)
			}
		})
	}
}

func TestRequestBytesOmitsHTTPErrorResponseBody(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		expectedAbsent []string
	}{
		{
			name:           "omits an upstream response body from returned errors",
			responseBody:   `{"message":"response-body-secret"}`,
			expectedAbsent: []string{"response-body-secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte(tt.responseBody))
				if err != nil {
					t.Errorf("expected response body write to succeed: %v", err)
				}
			}))
			defer server.Close()

			httpClient := HTTPClient{Client: server.Client()}
			_, err := httpClient.RequestBytes(t.Context(), HTTPRequestData{
				Method: http.MethodGet,
				URL:    server.URL + "/search?apiKey=status-query-secret",
			})
			if err == nil {
				t.Fatal("expected request to fail")
			}

			var statusErr HTTPStatusCodeError
			if !errors.As(err, &statusErr) {
				t.Fatalf("expected HTTP status error, got %T", err)
			}
			for _, secret := range append(tt.expectedAbsent, "status-query-secret") {
				if strings.Contains(err.Error(), secret) || strings.Contains(statusErr.URL, secret) || strings.Contains(statusErr.Message, secret) {
					t.Fatalf("expected HTTP error fields not to contain %q", secret)
				}
			}
			if !strings.Contains(statusErr.Message, "response body omitted") {
				t.Fatalf("expected response body summary, got %q", statusErr.Message)
			}
		})
	}
}
