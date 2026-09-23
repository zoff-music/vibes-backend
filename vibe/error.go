package vibe

// ErrorResponse is the JSON body returned for ordinary handler failures.
// Detailed internal errors are logged rather than exposed to callers.
// @Description Ordinary handler failures return an error message. Middleware may instead return plain-text errors. Explicitly propagated upstream errors can also include namespace, message, and propagate fields while retaining the upstream HTTP status.
type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}
