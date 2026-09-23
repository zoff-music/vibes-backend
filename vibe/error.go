package vibe

// ErrorResponse is the JSON body returned for ordinary handler failures.
// Detailed internal errors are logged rather than exposed to callers.
// @Description Ordinary handler failures return an error message. Middleware may instead return plain-text errors. Locally defined public errors can also include namespace, message, and propagate fields. Upstream and internal error details are never forwarded.
type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}
