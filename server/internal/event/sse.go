package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// RoomSSEEvents contains dependencies for the room event stream.
type RoomSSEEvents struct {
	Participants vibe.RoomEventParticipantStorage
	Notifier     vibe.RoomEventNotifier
	Subscriber   vibe.Subscriber
	Client       vibe.SSEClient
}

// AdminSSEEvents contains dependencies for the admin event stream.
type AdminSSEEvents struct {
	Rooms      vibe.AdminRoomLister
	Subscriber vibe.Subscriber
	Client     vibe.SSEClient
}

type sseEvents struct {
	Topic      string
	Connection sseConnection
	Subscriber vibe.Subscriber
}

type sseConnection interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context)
	Heartbeat(ctx context.Context) error
	InitialEvent(ctx context.Context) (*vibe.RoomEvent, error)
	ShouldSend(event vibe.RoomEvent) bool
}

type roomSSEConnection struct {
	Participants          vibe.RoomEventParticipantStorage
	Notifier              vibe.RoomEventNotifier
	RoomID                string
	UserID                string
	CastOwnerID           string
	IsCastReceiver        bool
	IsNewSession          bool
	ParticipantRegistered bool
}

type adminSSEConnection struct {
	Rooms vibe.AdminRoomLister
}

type connectedData struct {
	Time int64 `json:"time"`
}

// Events handles GET /api/v1/rooms/{id}/events.
//
//	@Summary	Subscribe to room events
//	@Tags		events
//	@Produce	text/event-stream
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{string}	string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/events [get]
func (e RoomSSEEvents) Events() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		connection := roomSSEConnection{
			Participants: e.Participants,
			Notifier:     e.Notifier,
			RoomID:       roomID,
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if ok {
			connection.UserID = session.UserID
			connection.IsCastReceiver = session.AuthType == "cast"
			connection.IsNewSession = session.IsNew
			if connection.IsCastReceiver {
				connection.CastOwnerID = session.UserID
			}
		}

		if connection.IsCastReceiver && connection.CastOwnerID != "" {
			connection.UserID = fmt.Sprintf("cast:%s:%s", roomID, connection.CastOwnerID)
		}

		events := sseEvents{
			Topic:      vibe.RoomTopic(roomID),
			Connection: &connection,
			Subscriber: e.Subscriber,
		}
		events.SubscribeAndListen(ctx, e.Client, w)
	}
}

// Events handles GET /api/v1/admin/events.
//
//	@Summary		Subscribe to room administration events
//	@Description	Streams the initial room list and subsequent room summary updates to authenticated administrators.
//	@Tags			admin
//	@Produce		text/event-stream
//	@Success		200	{string}	string
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/admin/events [get]
func (e AdminSSEEvents) Events() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		events := sseEvents{
			Topic: vibe.AdminTopic,
			Connection: &adminSSEConnection{
				Rooms: e.Rooms,
			},
			Subscriber: e.Subscriber,
		}
		events.SubscribeAndListen(ctx, e.Client, w)
	}
}

// SubscribeAndListen subscribes to an SSE topic and writes received events to the HTTP stream.
func (e sseEvents) SubscribeAndListen(ctx context.Context, client vibe.SSEClient, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}

	container, err := e.Subscriber.Subscribe(ctx, e.Topic)
	if err != nil {
		log.Printf("error subscribing to SSE topic %s in Events: %v", e.Topic, err)
		http.Error(w, "failed to subscribe to event stream", http.StatusInternalServerError)
		return
	}
	defer container.Subscription.Destroy()

	err = e.Connection.Connect(ctx)
	if err != nil {
		log.Printf("error connecting SSE topic %s in Events: %v", e.Topic, err)
	}
	defer e.Connection.Disconnect(context.WithoutCancel(ctx))

	payload, err := json.Marshal(connectedData{
		Time: time.Now().UnixMilli(),
	})
	if err != nil {
		log.Printf("error marshaling connected event in Events: %v", err)
		return
	}

	err = client.Write(w, flusher, vibe.RoomEvent{
		Type:    vibe.ConnectedEvent,
		Payload: payload,
	})
	if err != nil {
		log.Printf("error writing connected event in Events: %v", err)
		return
	}

	initialEvent, err := e.Connection.InitialEvent(ctx)
	if err != nil {
		log.Printf("error creating initial SSE event for topic %s in Events: %v", e.Topic, err)
	} else {
		err = client.Write(w, flusher, *initialEvent)
		if err != nil {
			log.Printf("error writing initial SSE event in Events: %v", err)
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
			err = e.Connection.Heartbeat(ctx)
			if err != nil {
				log.Printf("error handling SSE heartbeat for topic %s in Events: %v", e.Topic, err)
			}
			client.Heartbeat(w, flusher)
		case data, open := <-messages:
			if !open {
				return
			}

			var streamEvent vibe.RoomEvent
			err = json.Unmarshal(data, &streamEvent)
			if err != nil {
				log.Printf("error unmarshaling SSE event in Events: %v", err)
				continue
			}
			if !e.Connection.ShouldSend(streamEvent) {
				continue
			}

			err = client.Write(w, flusher, streamEvent)
			if err != nil {
				log.Printf("error writing SSE event in Events: %v", err)
				return
			}
		}
	}
}

func (c *roomSSEConnection) Connect(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Connect")
	defer span.End()

	if c.UserID == "" || c.IsNewSession {
		return nil
	}

	err := c.Participants.UpdateParticipant(
		ctx,
		c.RoomID,
		c.UserID,
		!c.IsCastReceiver,
		c.IsCastReceiver,
		c.CastOwnerID,
	)
	if err != nil {
		return fmt.Errorf("error updating participant in Connect: %w", err)
	}

	c.ParticipantRegistered = true
	err = c.notifyUsers(ctx)
	if err != nil {
		return fmt.Errorf("error notifying users in Connect: %w", err)
	}

	return nil
}

func (c *roomSSEConnection) Disconnect(ctx context.Context) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Disconnect")
	defer span.End()

	if !c.ParticipantRegistered {
		return
	}

	err := c.notifyUsers(ctx)
	if err != nil {
		log.Printf("error notifying users in Disconnect: %v", err)
	}
}

func (c *roomSSEConnection) Heartbeat(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Heartbeat")
	defer span.End()

	if c.UserID == "" {
		return nil
	}

	err := c.Participants.UpdateParticipant(
		ctx,
		c.RoomID,
		c.UserID,
		!c.IsCastReceiver,
		c.IsCastReceiver,
		c.CastOwnerID,
	)
	if err != nil {
		return fmt.Errorf("error updating participant in Heartbeat: %w", err)
	}
	if c.ParticipantRegistered {
		return nil
	}

	c.ParticipantRegistered = true
	err = c.notifyUsers(ctx)
	if err != nil {
		return fmt.Errorf("error notifying users in Heartbeat: %w", err)
	}

	return nil
}

func (c *roomSSEConnection) InitialEvent(ctx context.Context) (*vibe.RoomEvent, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "InitialEvent")
	defer span.End()

	state, err := c.Participants.GetPlaybackState(ctx, c.RoomID)
	if err != nil {
		return nil, fmt.Errorf("error fetching playback state in InitialEvent: %w", err)
	}

	if state.IsPlaying && state.UpdatedAt.Before(time.Now()) {
		state.PositionMs += int(time.Since(state.UpdatedAt).Milliseconds())
		state.UpdatedAt = time.Now()
	}
	state.ServerTimeMs = int(time.Now().UnixMilli())

	payload, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("error marshaling playback state in InitialEvent: %w", err)
	}

	event := vibe.RoomEvent{
		Type:    vibe.PlaybackUpdate,
		Payload: payload,
	}

	return &event, nil
}

func (c *roomSSEConnection) ShouldSend(event vibe.RoomEvent) bool {
	filterID := c.UserID
	if c.IsCastReceiver && c.CastOwnerID != "" {
		filterID = c.CastOwnerID
	}

	shouldSend := event.UserID == "" || event.UserID != filterID

	return shouldSend
}

func (c *roomSSEConnection) notifyUsers(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "notifyUsers")
	defer span.End()

	counts, err := c.Participants.GetActiveListenerCounts(ctx, c.RoomID, 15*time.Second)
	if err != nil {
		return fmt.Errorf("error fetching active participants in notifyUsers: %w", err)
	}

	count := counts.ActiveListeners
	if counts.ActiveListeners == 0 && counts.ActiveCastReceivers > 0 {
		count = 1
	}

	payload, err := json.Marshal(count)
	if err != nil {
		return fmt.Errorf("error marshaling active participant count in notifyUsers: %w", err)
	}

	err = c.Notifier.NotifyRoomUpdate(context.WithoutCancel(ctx), c.RoomID, vibe.RoomEvent{
		Type:    vibe.UsersUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying room update in notifyUsers: %w", err)
	}

	return nil
}

func (c *adminSSEConnection) Connect(ctx context.Context) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Connect")
	defer span.End()

	return nil
}

func (c *adminSSEConnection) Disconnect(ctx context.Context) {
	span, _ := tracing.StartSpanFromContext(ctx, "Disconnect")
	defer span.End()
}

func (c *adminSSEConnection) Heartbeat(ctx context.Context) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Heartbeat")
	defer span.End()

	return nil
}

func (c *adminSSEConnection) InitialEvent(ctx context.Context) (*vibe.RoomEvent, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "InitialEvent")
	defer span.End()

	rooms, err := c.Rooms.ListAdminRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing admin rooms in InitialEvent: %w", err)
	}

	payload, err := json.Marshal(rooms)
	if err != nil {
		return nil, fmt.Errorf("error marshaling admin rooms in InitialEvent: %w", err)
	}

	event := vibe.RoomEvent{
		Type:    vibe.AdminRoomsUpdate,
		Payload: payload,
	}

	return &event, nil
}

func (c *adminSSEConnection) ShouldSend(event vibe.RoomEvent) bool {
	return true
}
