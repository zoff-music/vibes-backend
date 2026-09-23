package client

import (
	"encoding/json/v2"
	"fmt"
)

type ErrorCodeResponseBody struct {
	Namespace string `json:"namespace"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Propagate bool   `json:"propagate,omitzero"`
}

// ErrorCodeWrapper carries a structured error between clients.
// HTTP handlers do not trust it as a public error; use vibe.PublicError there.
type ErrorCodeWrapper struct {
	Err          error
	ResponseBody ErrorCodeResponseBody
	StatusCode   int
}

func (e ErrorCodeWrapper) Error() string {
	return fmt.Sprintf("error propagatable: %v", e.Err)
}

func (e ErrorCodeWrapper) Unwrap() error {
	return e.Err
}

func (e ErrorCodeWrapper) GetResponseBody() ([]byte, error) {
	// We don't want to overwrite the namespace
	// if we are already propagating something here
	if e.ResponseBody.Namespace == "" {
		e.ResponseBody.Namespace = applicationName
	}

	resp, err := json.Marshal(&e.ResponseBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling propagatable error response body: %w", err)
	}

	return resp, nil
}
