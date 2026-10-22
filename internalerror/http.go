package internalerror

import "fmt"

// HTTPStatusCodeError is an error that occurs when receiving an unexpected status
// code (>= 400).
type HTTPStatusCodeError struct {
	URL        string
	StatusCode int
	Message    string
	// ResponseBody is for provider-specific classification, never logging or propagation.
	ResponseBody []byte `json:"-"`
}

// Error return an error string.
func (e HTTPStatusCodeError) Error() string {
	result := fmt.Sprintf("error response from %s, got status: %d. Message: %s", e.URL, e.StatusCode, e.Message)
	return result
}
