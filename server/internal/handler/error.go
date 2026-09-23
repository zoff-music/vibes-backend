package handler

import (
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"

	"github.com/zoff-music/vibes-backend/vibe"
)

func handleError(w http.ResponseWriter, err error, statusCode int, shouldLog bool) {
	if shouldLog {
		log.Println(err.Error())
	}

	errorBody, _ := json.Marshal(vibe.ErrorResponse{Error: "something went wrong"})
	var publicError vibe.PublicError
	if errors.As(err, &publicError) {
		response, responseErr := publicError.Response()
		if responseErr != nil {
			log.Println(responseErr.Error())
		} else if publicError.StatusCode >= 400 && publicError.StatusCode <= 599 {
			body, marshalErr := json.Marshal(response)
			if marshalErr != nil {
				log.Println(marshalErr.Error())
			} else {
				w.Header().Set("X-preserve-error", "1")
				statusCode = publicError.StatusCode
				errorBody = body
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(errorBody)
}
