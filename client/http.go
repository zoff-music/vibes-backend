// Package client contains an HTTP client.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	opentracing "github.com/zoff-music/vibes-backend/monitoring/opentracing"
)

// Parameters provides the parameters used when creating a new HTTP client.
type Parameters struct {
	Timeout *time.Duration
}

// NewHTTPClient instantiates a new HTTPClient based on provided parameters.
func NewHTTPClient(parameters Parameters) HTTPClient {
	if parameters.Timeout == nil {
		timeout := 1 * time.Second
		parameters.Timeout = &timeout
	}

	client := &http.Client{
		Timeout: *parameters.Timeout,
	}

	return HTTPClient{client}
}

// HTTPRequestData contains the request data.
type HTTPRequestData struct {
	Method    string
	URL       string
	Headers   map[string]string
	Body      []byte
	Payload   *url.Values
	BasicAuth *BasicAuth
}

// BasicAuth contains crendentials for basic authentication.
type BasicAuth struct {
	Username string
	Password string
}

// HTTPClient contains the HTTP client.
type HTTPClient struct {
	*http.Client
}

// HTTPStatusCodeError is an error that occurs when receiving an unexpected status
// code (>= 400).
type HTTPStatusCodeError struct {
	URL        string
	StatusCode int
	Message    string
}

// Error return an error string.
func (e HTTPStatusCodeError) Error() string {
	return fmt.Sprintf("error response from %s, got status: %d. Message: %s", e.URL, e.StatusCode, e.Message)
}

// RequestBytes does the actual HTTP request.
// Returns a slice of bytes or an error.
func (client *HTTPClient) RequestBytes(ctx context.Context, reqData HTTPRequestData) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RequestBytes")
	defer span.Finish()

	r, err := client.request(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Printf("error closing response body: %s", err.Error())
		}
	}()

	if r.StatusCode >= 400 {
		resp, _ := io.ReadAll(r.Body)

		message := string(resp)
		span.SetBoolTag("error", true)
		span.LogFields(
			opentracing.SpanField{
				Key:  "message",
				Type: opentracing.SpanFieldError,
				ErrorValue: fmt.Errorf(
					"error making request to %s, body: %s. Got error: %s",
					reqData.URL,
					redactBodyForLog(reqData.Headers, reqData.Body),
					message,
				),
			},
		)

		httpStatusCodeError := HTTPStatusCodeError{
			URL:        reqData.URL,
			StatusCode: r.StatusCode,
			Message:    message,
		}

		// Check if error should propagate
		var errorCodeWrapper ErrorCodeResponseBody
		_ = json.Unmarshal(resp, &errorCodeWrapper)
		if errorCodeWrapper.Propagate {
			return nil, ErrorCodeWrapper{
				Err:          httpStatusCodeError,
				ResponseBody: errorCodeWrapper,
				StatusCode:   r.StatusCode,
			}
		}

		return nil, httpStatusCodeError
	}

	return io.ReadAll(r.Body)
}

func (client *HTTPClient) request(ctx context.Context, reqData HTTPRequestData) (*http.Response, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "request")
	defer span.Finish()

	req, err := http.NewRequest(reqData.Method, reqData.URL, bytes.NewBuffer(reqData.Body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if reqData.Payload != nil {
		req.URL.RawQuery = reqData.Payload.Encode()
	}

	parentSpan := opentracing.SpanFromContext(ctx)

	if parentSpan != nil {
		err := opentracing.GlobalTracer().Inject(
			parentSpan.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header),
		)
		if err != nil {
			log.Printf("error injecting span into request headers: %s", err.Error())
		}
	}

	for k, v := range reqData.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("User-Agent", "template-api-go")

	if reqData.BasicAuth != nil {
		req.SetBasicAuth(reqData.BasicAuth.Username, reqData.BasicAuth.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		if reqData.Method == http.MethodPost {
			return resp, fmt.Errorf("error making request: %w. Body: %s", err, redactBodyForLog(reqData.Headers, reqData.Body))
		}

		return resp, fmt.Errorf("error making request: %w. Query: %v", err, req.URL.RawQuery)
	}

	return resp, nil
}

// redactBodyForLog returns a redacted representation of a request body suitable for logs/traces.
// It tries to parse JSON and form bodies and redact common secret/token fields; otherwise it
// returns a size-only placeholder.
func redactBodyForLog(headers map[string]string, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	contentType := ""
	for k, v := range headers {
		if strings.EqualFold(k, "Content-Type") {
			contentType = v
			break
		}
	}

	redactKey := func(k string) bool {
		switch strings.ToLower(k) {
		case "access_token", "refresh_token", "id_token", "token",
			"client_secret", "clientsecret", "secret", "password",
			"authorization", "code", "api_key", "apikey":
			return true
		default:
			return false
		}
	}

	if strings.Contains(contentType, "application/json") {
		var obj map[string]json.RawMessage
		err := json.Unmarshal(body, &obj)
		if err == nil {
			for k := range obj {
				if redactKey(k) {
					obj[k] = json.RawMessage(`"[REDACTED]"`)
				}
			}
			b, err := json.Marshal(obj)
			if err == nil {
				return string(b)
			}
		}
		return fmt.Sprintf("[unparseable json body: %d bytes]", len(body))
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err == nil {
			for k := range values {
				if redactKey(k) {
					values.Set(k, "[REDACTED]")
				}
			}
			return values.Encode()
		}
		return fmt.Sprintf("[unparseable form body: %d bytes]", len(body))
	}

	return fmt.Sprintf("[body omitted: %d bytes]", len(body))
}
