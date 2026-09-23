package handler

import (
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"

	"github.com/zoff-music/vibes-backend/vibe"
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

	w.Header().Set("Content-Type", "application/json")

	var publicError vibe.PublicError
	if errors.As(err, &publicError) && publicError.StatusCode >= 400 && publicError.StatusCode <= 599 {
		response, responseErr := publicError.Response()
		if responseErr != nil {
			log.Println(responseErr.Error())
		}

		if responseErr == nil {
			body, marshalErr := json.Marshal(response)
			if marshalErr != nil {
				log.Println(marshalErr.Error())
			}

			if marshalErr == nil {
				w.Header().Set("X-preserve-error", "1")
				w.WriteHeader(publicError.StatusCode)
				_, _ = w.Write(body)
				return
			}
		}
	}

	body, marshalErr := json.Marshal(vibe.ErrorResponse{
		Error: "something went wrong",
	})
	if marshalErr != nil {
		log.Println(marshalErr.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}
