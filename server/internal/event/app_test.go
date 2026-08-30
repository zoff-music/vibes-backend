package event

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/client/database"
	redisclient "github.com/zoff-music/vibes-backend/client/redis"
	"github.com/zoff-music/vibes-backend/client/soundcloud"
	"github.com/zoff-music/vibes-backend/client/youtube"
	"github.com/zoff-music/vibes-backend/internalerror"
)

type appEventNextRateTestCase struct {
	name     string
	idleRate time.Duration
	err      error
	want     time.Duration
}

type appEventConfigurationTestCase struct {
	name      string
	index     int
	eventName string
	rate      time.Duration
	idleRate  time.Duration
}

func TestAppEventNextRate(t *testing.T) {
	rate := 100 * time.Millisecond
	idleRate := 250 * time.Millisecond
	expectedErr := internalerror.ErrExpected{
		Err: internalerror.ErrNonRecoverable{
			Err: errors.New("no queued work"),
		},
	}

	tests := []appEventNextRateTestCase{
		{
			name:     "expected idle result uses idle rate",
			idleRate: idleRate,
			err:      expectedErr,
			want:     idleRate,
		},
		{
			name:     "wrapped expected idle result uses idle rate",
			idleRate: idleRate,
			err:      fmt.Errorf("error processing scheduled event: %w", expectedErr),
			want:     idleRate,
		},
		{
			name:     "successful result uses active rate",
			idleRate: idleRate,
			want:     rate,
		},
		{
			name:     "unexpected error uses active rate",
			idleRate: idleRate,
			err:      errors.New("database unavailable"),
			want:     rate,
		},
		{
			name: "missing idle rate uses active rate",
			err:  expectedErr,
			want: rate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := AppEvent{
				Rate:     rate,
				IdleRate: tt.idleRate,
			}

			got := event.nextRate(tt.err)
			if got != tt.want {
				t.Fatalf("nextRate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetAppEventsRates(t *testing.T) {
	tests := []appEventConfigurationTestCase{
		{
			name:      "playlist imports back off while idle",
			index:     0,
			eventName: "ImportPlaylistSong",
			rate:      100 * time.Millisecond,
			idleRate:  250 * time.Millisecond,
		},
	}

	events := GetAppEvents(
		&database.Client{},
		&redisclient.Client{},
		&soundcloud.Client{},
		&youtube.Client{},
		nil,
		[]string{},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(events) <= tt.index {
				t.Fatalf("GetAppEvents() returned %d events, want index %d", len(events), tt.index)
			}

			event := events[tt.index]
			if event.Name != tt.eventName {
				t.Fatalf("event name = %q, want %q", event.Name, tt.eventName)
			}
			if event.Rate != tt.rate {
				t.Fatalf("event rate = %s, want %s", event.Rate, tt.rate)
			}
			if event.IdleRate != tt.idleRate {
				t.Fatalf("event idle rate = %s, want %s", event.IdleRate, tt.idleRate)
			}
		})
	}
}
