package handler

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

// RoomEventStream creates room event connections.
type RoomEventStream struct {
	Participants vibe.RoomEventParticipantStorage
	Notifier     vibe.RoomEventNotifier
}

// AdminEventStream creates admin event connections.
type AdminEventStream struct {
	Rooms vibe.AdminRoomLister
}

type roomEventConnection struct {
	Participants          vibe.RoomEventParticipantStorage
	Notifier              vibe.RoomEventNotifier
	RoomID                string
	UserID                string
	CastOwnerID           string
	IsCastReceiver        bool
	IsNewSession          bool
	ParticipantRegistered bool
}

type adminEventConnection struct {
	Rooms vibe.AdminRoomLister
}

// Open creates a request-scoped room event stream.
func (h RoomEventStream) Open(r *http.Request) (*vibe.SSEStream, error) {
	roomID := mux.Vars(r)["id"]
	connection := roomEventConnection{
		Participants: h.Participants,
		Notifier:     h.Notifier,
		RoomID:       roomID,
	}

	session, ok := helper.GetSessionFromContext(r.Context())
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

	stream := vibe.SSEStream{
		Topic:      vibe.RoomTopic(roomID),
		Connection: &connection,
	}

	return &stream, nil
}

// Open creates an admin event stream.
func (h AdminEventStream) Open(_ *http.Request) (*vibe.SSEStream, error) {
	stream := vibe.SSEStream{
		Topic: vibe.AdminTopic,
		Connection: &adminEventConnection{
			Rooms: h.Rooms,
		},
	}

	return &stream, nil
}

func (c *roomEventConnection) Connect(ctx context.Context) error {
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

func (c *roomEventConnection) Disconnect(ctx context.Context) {
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

func (c *roomEventConnection) Heartbeat(ctx context.Context) error {
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

func (c *roomEventConnection) InitialEvent(ctx context.Context) (*vibe.StreamEvent, error) {
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

	event := vibe.StreamEvent{
		Type:    vibe.PlaybackUpdate,
		Payload: payload,
	}

	return &event, nil
}

func (c *roomEventConnection) ShouldSend(event vibe.StreamEvent) bool {
	filterID := c.UserID
	if c.IsCastReceiver && c.CastOwnerID != "" {
		filterID = c.CastOwnerID
	}

	shouldSend := event.UserID == "" || event.UserID != filterID

	return shouldSend
}

func (c *roomEventConnection) notifyUsers(ctx context.Context) error {
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

func (c *adminEventConnection) Connect(ctx context.Context) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Connect")
	defer span.End()

	return nil
}

func (c *adminEventConnection) Disconnect(ctx context.Context) {
	span, _ := tracing.StartSpanFromContext(ctx, "Disconnect")
	defer span.End()
}

func (c *adminEventConnection) Heartbeat(ctx context.Context) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Heartbeat")
	defer span.End()

	return nil
}

func (c *adminEventConnection) InitialEvent(ctx context.Context) (*vibe.StreamEvent, error) {
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

	event := vibe.StreamEvent{
		Type:    vibe.AdminRoomsUpdate,
		Payload: payload,
	}

	return &event, nil
}

func (c *adminEventConnection) ShouldSend(vibe.StreamEvent) bool {
	return true
}
