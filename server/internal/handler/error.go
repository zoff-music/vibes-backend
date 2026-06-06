package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/zoff-music/vibes/client"
)

func handleError(
	w http.ResponseWriter,
	err error,
	statusCode int,
	shouldLog bool,
) {
	if shouldLog {
		log.Println(err.Error())
	}

	errorBody, _ := json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: "something went wrong",
	})

	var errorCodeWrapper client.ErrorCodeWrapper
	if errors.As(err, &errorCodeWrapper) {
		w.Header().Add("X-preserve-error", "1")

		statusCode = errorCodeWrapper.StatusCode
		errorBody, err = errorCodeWrapper.GetResponseBody()
		if err != nil {
			log.Println(err.Error())
		}
	}

	w.WriteHeader(statusCode)
	w.Write(errorBody)
}
