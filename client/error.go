package client

import (
	"encoding/json"
	"fmt"
)

type ErrorCodeResponseBody struct {
	Namespace string `json:"namespace"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Propagate bool   `json:"propagate,omitempty"`
}

// ErrorCodeWrapper is an error type that we want to
// send downstream (can both be one step, but also fully propagated)
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
		e.ResponseBody.Namespace = "template-api-go"
	}
	resp, err := json.Marshal(&e.ResponseBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling propagatable error response body: %w", err)
	}
	return resp, nil
}
