package event

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

// SSEEvents contains the named SSE event streams.
type SSEEvents []SSEEvent

// SSEEvent describes a named SSE event stream.
type SSEEvent struct {
	Name       string
	Subscriber vibe.Subscriber
	Handler    vibe.SSEHandler
}

type connectedData struct {
	Time int64 `json:"time"`
}

// Events handles named SSE event stream routes.
//
//	@Summary	Subscribe to events
//	@Tags		events
//	@Produce	text/event-stream
//	@Success	200	{string}	string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/events [get]
//	@Router		/api/v1/admin/events [get]
func (e SSEEvents) Events(client vibe.SSEClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		if route == nil {
			http.Error(w, "event route is not configured", http.StatusInternalServerError)
			return
		}

		name := route.GetName()
		var namedEvent SSEEvent
		for _, event := range e {
			if event.Name == name {
				namedEvent = event
				break
			}
		}
		if namedEvent.Handler == nil {
			http.Error(w, "event route is not registered", http.StatusInternalServerError)
			return
		}

		stream, err := namedEvent.Handler.Open(r)
		if err != nil {
			log.Printf("error opening SSE event %s in Events: %v", namedEvent.Name, err)
			http.Error(w, "failed to open event stream", http.StatusInternalServerError)
			return
		}
		namedEvent.SubscribeAndListen(r.Context(), client, w, stream)
	}
}

// SubscribeAndListen subscribes to an SSE topic and writes received events to the HTTP stream.
func (e SSEEvent) SubscribeAndListen(
	ctx context.Context,
	client vibe.SSEClient,
	w http.ResponseWriter,
	stream *vibe.SSEStream,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}

	container, err := e.Subscriber.Subscribe(ctx, stream.Topic)
	if err != nil {
		log.Printf("error subscribing to SSE event %s in Events: %v", e.Name, err)
		http.Error(w, "failed to subscribe to event stream", http.StatusInternalServerError)
		return
	}
	defer container.Subscription.Destroy()

	err = stream.Connection.Connect(ctx)
	if err != nil {
		log.Printf("error connecting SSE event %s in Events: %v", e.Name, err)
	}
	defer stream.Connection.Disconnect(context.WithoutCancel(ctx))

	payload, err := json.Marshal(connectedData{
		Time: time.Now().UnixMilli(),
	})
	if err != nil {
		log.Printf("error marshaling connected event in Events: %v", err)
		return
	}

	err = client.Write(w, flusher, vibe.StreamEvent{
		Type:    vibe.ConnectedEvent,
		Payload: payload,
	})
	if err != nil {
		log.Printf("error writing connected event in Events: %v", err)
		return
	}

	initialEvent, err := stream.Connection.InitialEvent(ctx)
	if err != nil {
		log.Printf("error creating initial SSE event %s in Events: %v", e.Name, err)
	} else {
		err = client.Write(w, flusher, *initialEvent)
		if err != nil {
			log.Printf("error writing initial SSE event %s in Events: %v", e.Name, err)
			return
		}
	}

	ticker := time.NewTicker(client.HeartbeatRate())
	defer ticker.Stop()

	messages := container.Subscription.Listen()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err = stream.Connection.Heartbeat(ctx)
			if err != nil {
				log.Printf("error handling SSE heartbeat %s in Events: %v", e.Name, err)
			}
			client.Heartbeat(w, flusher)
		case data, open := <-messages:
			if !open {
				return
			}

			var streamEvent vibe.StreamEvent
			err = json.Unmarshal(data, &streamEvent)
			if err != nil {
				log.Printf("error unmarshaling SSE event %s in Events: %v", e.Name, err)
				continue
			}
			if !stream.Connection.ShouldSend(streamEvent) {
				continue
			}

			err = client.Write(w, flusher, streamEvent)
			if err != nil {
				log.Printf("error writing SSE event %s in Events: %v", e.Name, err)
				return
			}
		}
	}
}
