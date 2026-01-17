package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zoff-music/vibes/client"

	log "github.com/sirupsen/logrus"
)

func handleError(
	w http.ResponseWriter,
	err error,
	statusCode int,
	shouldLog bool,
) {
	if shouldLog {
		log.Error(err.Error())
	}

	errorBody, _ := json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: err.Error(), // Change to something generic if this API is publicly exposed
	})

	var errorCodeWrapper client.ErrorCodeWrapper
	if errors.As(err, &errorCodeWrapper) {
		w.Header().Add("X-preserve-error", "1")

		statusCode = errorCodeWrapper.StatusCode
		errorBody, err = errorCodeWrapper.GetResponseBody()
		if err != nil {
			log.Error(err.Error())
		}
	}

	w.WriteHeader(statusCode)
	w.Write(errorBody)
}
