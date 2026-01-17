package handler

import (
	"encoding/json"
	"fmt"
	"github.com/zoff-music/cibes/example"
	"net/http"
)

// Example is handler that provides an example of how handlers should be written.
// 		GET /api/v1/api
// 		Responds: 200, 500
//
// The handler should accept an interface(s), and should contain only high level
// business logic. There should be no implementation details here (except I guess
// stuff specific to http, like writing the response).
func Example(
	fetcher example.DataFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get example data
		exampleData, err := fetcher.GetExampleData(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting example data in example handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Marshal data and respond
		response, err := json.Marshal(&exampleData)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshalling example data in example handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}
