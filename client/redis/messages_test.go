package redis

import (
	"context"
	"encoding/json/v2"
	"os"
	"strings"
	"testing"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestMessageRetention(t *testing.T) {
	url := os.Getenv("VIBES_TEST_REDIS_URL")
	if url == "" {
		t.Skip("set VIBES_TEST_REDIS_URL to an isolated local Redis")
	}
	if !strings.HasPrefix(url, "redis://127.0.0.1:") {
		t.Fatal("integration Redis must be local")
	}
	tests := []struct {
		name    string
		maximum int
	}{{name: "bounded chat replay", maximum: 2}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			err := client.Init(context.Background(), &config.Config{RedisURL: url, RoomEventReplayMaxEvents: tt.maximum, RoomEventReplayMaxAge: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			roomID := uuid.NewString()
			for index := 0; index < 3; index++ {
				payload, marshalErr := json.Marshal(vibe.RoomMessage{ID: uuid.NewString(), Name: "Mia", Kind: "chat", Text: "retained", CreatedAt: time.Now().UnixMilli()})
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				err = client.NotifyRoomUpdate(context.Background(), roomID, vibe.RoomEvent{Type: vibe.MessageEvent, Payload: payload})
				if err != nil {
					t.Fatal(err)
				}
			}
			connection := client.Redis.Get()
			defer connection.Close()
			key := replayKey("chat:" + roomID)
			count, err := redigo.Int(connection.Do("XLEN", key))
			if err != nil {
				t.Fatal(err)
			}
			if count != tt.maximum {
				t.Fatalf("chat has %d entries, want %d", count, tt.maximum)
			}
			count, err = redigo.Int(connection.Do("EXISTS", replayKey(roomTopicName(roomID))))
			if err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("chat leaked into playback stream")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			replay, err := client.SubscribeFrom(ctx, "chat:"+roomID, "0-0")
			if err != nil {
				t.Fatal(err)
			}
			ids := []string{}
			for index := 0; index < tt.maximum; index++ {
				select {
				case data := <-replay.Subscription.Listen():
					var event vibe.RoomEvent
					err = json.Unmarshal(data, &event)
					if err != nil {
						t.Fatal(err)
					}
					if event.Type != vibe.MessageEvent || event.ID == "" {
						t.Fatalf("bad message replay: %+v", event)
					}
					ids = append(ids, event.ID)
				case <-ctx.Done():
					t.Fatal("replay timed out")
				}
			}
			replay.Subscription.Destroy()
			if ids[0] == ids[1] {
				t.Fatal("stream cursors must be distinct")
			}
			time.Sleep(1100 * time.Millisecond)
			count, err = redigo.Int(connection.Do("EXISTS", key))
			if err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("idle chat history did not expire")
			}
		})
	}
}
