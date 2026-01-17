package example

import (
	"context"
	"time"
)

// Data contains example data indicating if the data is fake
// or not.
type Data struct {
	IsFake bool      `json:"isFake"`
	Date   time.Time `json:"date"`
}

// DataFetcher is an interface for getting example data.
type DataFetcher interface {
	GetExampleData(ctx context.Context) (*Data, error)
}

// DataRecorder is an interface for recording example data.
type DataRecorder interface {
	RecordExampleData(ctx context.Context, exampleData Data) error
}

// DataNotifier is an interface for notifying other apps about
// example data that was recorded.
type DataNotifier interface {
	NotifyExampleData(ctx context.Context, exampleData Data) error
}
