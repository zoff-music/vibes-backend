package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

const statusClientClosedRequest = 499

func handleError(
	w http.ResponseWriter,
	err error,
	statusCode int,
	shouldLog bool,
) {
	if errors.Is(err, context.Canceled) {
		w.WriteHeader(statusClientClosedRequest)
		return
	}

	if shouldLog {
		log.Println(strconv.Quote(err.Error()))
	}

	w.Header().Set("Content-Type", "application/json")

	fallback, marshalErr := json.Marshal(vibe.ErrorResponse{
		Error: "something went wrong",
	})
	if marshalErr != nil {
		log.Println(marshalErr.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var publicError internalerror.ErrorCodeWrapper
	if !errors.As(err, &publicError) ||
		!publicError.ResponseBody.Propagate ||
		publicError.StatusCode < 400 ||
		publicError.StatusCode > 599 {
		w.WriteHeader(statusCode)
		_, _ = w.Write(fallback)
		return
	}

	body, marshalErr := publicError.GetResponseBody()
	if marshalErr != nil {
		log.Println(marshalErr.Error())

		w.WriteHeader(statusCode)
		_, _ = w.Write(fallback)
		return
	}

	w.Header().Set("X-preserve-error", "1")
	w.WriteHeader(publicError.StatusCode)
	_, _ = w.Write(body)
}
