// Package client contains an HTTP client.
package client

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// InstrumentedTransport provides the traced transport used by external HTTP clients.
func InstrumentedTransport() http.RoundTripper {
	return otelhttp.NewTransport(http.DefaultTransport)
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
	span, ctx := tracing.StartSpanFromContext(ctx, "RequestBytes")
	defer span.End()

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

		requestURL := redactURLForLog(r.Request.URL)
		message := fmt.Sprintf("[response body omitted: %d bytes]", len(resp))
		requestErr := fmt.Errorf(
			"error making request to %s, body: %s. Got response: %s",
			requestURL,
			redactBodyForLog(reqData.Headers, reqData.Body),
			message,
		)
		span.RecordError(requestErr)
		span.SetStatus(codes.Error, "error making request")
		span.SetAttributes(attribute.String("message", requestErr.Error()))

		httpStatusCodeError := HTTPStatusCodeError{
			URL:        requestURL,
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body in RequestBytes: %w", err)
	}

	return body, nil
}

func (client *HTTPClient) request(ctx context.Context, reqData HTTPRequestData) (*http.Response, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "request")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, reqData.Method, reqData.URL, bytes.NewBuffer(reqData.Body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if reqData.Payload != nil {
		req.URL.RawQuery = reqData.Payload.Encode()
	}

	for k, v := range reqData.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("User-Agent", applicationName)

	if reqData.BasicAuth != nil {
		req.SetBasicAuth(reqData.BasicAuth.Username, reqData.BasicAuth.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		redactedErr := redactHTTPError(err)
		return resp, fmt.Errorf(
			"error making request to %s: %w. Body: %s",
			redactURLForLog(req.URL),
			redactedErr,
			redactBodyForLog(reqData.Headers, reqData.Body),
		)
	}

	return resp, nil
}

const applicationName = "vibes-backend"

const maxLoggedValueBytes = 4096

const redactedValue = "[REDACTED]"

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

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Sprintf("[body omitted: %d bytes]", len(body))
	}
	mediaType = strings.ToLower(mediaType)

	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		redacted := redactJSONForLog(body)
		return redacted
	}

	if mediaType == "application/x-www-form-urlencoded" {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return fmt.Sprintf("[unparseable form body: %d bytes]", len(body))
		}
		redacted := redactValuesForLog(values)
		return redacted
	}

	return fmt.Sprintf("[body omitted: %d bytes]", len(body))
}

func redactJSONForLog(body []byte) string {
	value := jsontext.Value(body)
	if !value.IsValid() || (value.Kind() != '{' && value.Kind() != '[') {
		return fmt.Sprintf("[unparseable json body: %d bytes]", len(body))
	}

	redacted, err := redactJSONValue(value)
	if err != nil {
		return fmt.Sprintf("[unparseable json body: %d bytes]", len(body))
	}
	if len(redacted) > maxLoggedValueBytes {
		return fmt.Sprintf("[redacted json body omitted: %d bytes]", len(body))
	}

	return string(redacted)
}

func redactJSONValue(value jsontext.Value) (jsontext.Value, error) {
	switch value.Kind() {
	case '{':
		object := map[string]jsontext.Value{}
		err := json.Unmarshal(value, &object)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling JSON object for log redaction: %w", err)
		}
		for key, item := range object {
			if isSensitiveLogKey(key) {
				object[key] = jsontext.Value(`"[REDACTED]"`)
				continue
			}
			redacted, err := redactJSONValue(item)
			if err != nil {
				return nil, fmt.Errorf("error redacting JSON object value for log redaction: %w", err)
			}
			object[key] = redacted
		}
		body, err := json.Marshal(object)
		if err != nil {
			return nil, fmt.Errorf("error marshaling JSON object for log redaction: %w", err)
		}
		return jsontext.Value(body), nil
	case '[':
		items := []jsontext.Value{}
		err := json.Unmarshal(value, &items)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling JSON array for log redaction: %w", err)
		}
		for index, item := range items {
			redacted, err := redactJSONValue(item)
			if err != nil {
				return nil, fmt.Errorf("error redacting JSON array value for log redaction: %w", err)
			}
			items[index] = redacted
		}
		body, err := json.Marshal(items)
		if err != nil {
			return nil, fmt.Errorf("error marshaling JSON array for log redaction: %w", err)
		}
		return jsontext.Value(body), nil
	default:
		cloned := value.Clone()
		return cloned, nil
	}
}

func redactValuesForLog(values url.Values) string {
	redacted := make(url.Values, len(values))
	for key, items := range values {
		if isSensitiveLogKey(key) {
			redacted.Set(key, redactedValue)
			continue
		}
		copied := make([]string, len(items))
		copy(copied, items)
		redacted[key] = copied
	}

	encoded := redacted.Encode()
	if len(encoded) > maxLoggedValueBytes {
		return fmt.Sprintf("query_omitted=%d_bytes", len(encoded))
	}
	return encoded
}

func redactURLForLog(value *url.URL) string {
	if value == nil {
		return "[URL omitted]"
	}

	redacted := *value
	redacted.RawQuery = redactValuesForLog(value.Query())
	redacted.Fragment = ""
	redacted.RawFragment = ""
	if value.User != nil {
		redacted.User = url.User(redactedValue)
	}
	return redacted.String()
}

func redactHTTPError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return fmt.Errorf("error sending HTTP request: %w", err)
	}

	inner := redactHTTPError(urlErr.Err)
	redacted := &url.Error{
		Op:  urlErr.Op,
		URL: redactRawURLForLog(urlErr.URL),
		Err: inner,
	}
	return redacted
}

func redactRawURLForLog(rawURL string) string {
	value, err := url.Parse(rawURL)
	if err != nil {
		return "[URL omitted]"
	}
	redacted := redactURLForLog(value)
	return redacted
}

func isSensitiveLogKey(key string) bool {
	normalized := strings.Map(func(value rune) rune {
		if unicode.IsLetter(value) || unicode.IsDigit(value) {
			return unicode.ToLower(value)
		}
		return -1
	}, key)

	switch normalized {
	case "authorization", "code", "codeverifier", "key", "apikey",
		"pairingcode", "sessionid", "state":
		return true
	}

	return strings.Contains(normalized, "password") ||
		strings.HasSuffix(normalized, "token") ||
		strings.HasSuffix(normalized, "secret") ||
		strings.HasSuffix(normalized, "credential") ||
		strings.HasSuffix(normalized, "cookie") ||
		strings.HasSuffix(normalized, "privatekey") ||
		strings.HasSuffix(normalized, "signingkey") ||
		strings.HasSuffix(normalized, "encryptionkey")
}
