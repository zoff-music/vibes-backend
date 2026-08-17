package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

type eventStreams struct {
	pool      *redigo.Pool
	maxEvents int
	maxAge    time.Duration
}

func (c *Client) NotifyAdminUpdate(ctx context.Context, event vibe.AdminEvent) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "NotifyAdminUpdate")
	defer span.End()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling admin event in NotifyAdminUpdate: %w", err)
	}

	err = c.appendEvent(ctx, adminTopicName, data)
	if err != nil {
		return fmt.Errorf("error appending admin event in NotifyAdminUpdate: %w", err)
	}

	return nil
}

func (c *Client) NotifyRemoteUpdate(
	ctx context.Context,
	remoteID string,
	event vibe.RemoteEvent,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "NotifyRemoteUpdate")
	defer span.End()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling remote event in NotifyRemoteUpdate: %w", err)
	}

	err = c.appendEvent(ctx, remoteTopicName(remoteID), data)
	if err != nil {
		return fmt.Errorf("error appending remote event in NotifyRemoteUpdate: %w", err)
	}

	return nil
}

func (c *Client) NotifyRoomUpdate(
	ctx context.Context,
	roomID string,
	event vibe.RoomEvent,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "NotifyRoomUpdate")
	defer span.End()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling room event in NotifyRoomUpdate: %w", err)
	}

	err = c.appendRoomEvent(ctx, roomTopicName(roomID), data)
	if err != nil {
		return fmt.Errorf("error appending room event in NotifyRoomUpdate: %w", err)
	}

	return nil
}

func (c *Client) NotifyRoomUpdates(
	ctx context.Context,
	roomID string,
	events []vibe.RoomEvent,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "NotifyRoomUpdates")
	defer span.End()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("error marshaling room event in NotifyRoomUpdates: %w", err)
		}

		err = c.appendRoomEvent(ctx, roomTopicName(roomID), data)
		if err != nil {
			return fmt.Errorf("error notifying room event batch in NotifyRoomUpdates: %w", err)
		}
	}

	return nil
}

func (c *Client) appendRoomEvent(ctx context.Context, topicName string, data []byte) error {
	err := c.appendEvent(ctx, topicName, data)
	if err != nil {
		return fmt.Errorf("error appending room event: %w", err)
	}

	return nil
}

func (c *Client) appendEvent(ctx context.Context, topicName string, data []byte) error {
	if c.events == nil {
		return fmt.Errorf("error appending event: streams are not initialized")
	}

	err := c.events.append(ctx, topicName, data)
	if err != nil {
		return fmt.Errorf("error appending event: %w", err)
	}

	return nil
}

func (c *Client) PrepareReplay(
	ctx context.Context,
	topicName string,
	lastEventID string,
) (*vibe.ReplaySubscription, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "PrepareReplay")
	defer span.End()

	if c.events == nil {
		return nil, fmt.Errorf("error preparing room event replay: replay is not initialized")
	}

	replay, err := c.events.prepare(ctx, topicName, lastEventID)
	if err != nil {
		return nil, fmt.Errorf("error preparing room event replay: %w", err)
	}

	return replay, nil
}

func (c *Client) SubscribeFrom(
	ctx context.Context,
	topicName string,
	afterID string,
) (*vibe.SubscriptionContainer, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SubscribeFrom")
	defer span.End()

	if c.events == nil {
		return nil, fmt.Errorf("error subscribing to room events: replay is not initialized")
	}

	subscription, err := c.events.subscribe(ctx, topicName, afterID, true)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to room events: %w", err)
	}

	return &vibe.SubscriptionContainer{Subscription: subscription}, nil
}

func (c *Client) Subscribe(
	ctx context.Context,
	topicName string,
) (*vibe.SubscriptionContainer, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Subscribe")
	defer span.End()

	if c.events == nil {
		return nil, fmt.Errorf("error subscribing to events: streams are not initialized")
	}

	replay, err := c.events.prepare(ctx, topicName, "")
	if err != nil {
		return nil, fmt.Errorf("error preparing events in Subscribe: %w", err)
	}

	subscription, err := c.events.subscribe(
		ctx,
		topicName,
		replay.AfterID,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to events in Subscribe: %w", err)
	}

	return &vibe.SubscriptionContainer{Subscription: subscription}, nil
}

func (r *eventStreams) append(ctx context.Context, topicName string, data []byte) error {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := r.pool.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in append: %w", err)
	}
	defer connection.Close()

	key := replayKey(topicName)
	_, err = redigo.DoContext(
		connection,
		cctx,
		"XADD",
		key,
		"MAXLEN",
		r.maxEvents,
		"*",
		replayEventField,
		data,
	)
	if err != nil {
		return fmt.Errorf("error appending redis stream event in append: %w", err)
	}

	cutoffID := replayCutoffID(time.Now(), r.maxAge)
	_, err = redigo.DoContext(
		connection,
		cctx,
		"XTRIM",
		key,
		"MINID",
		cutoffID,
	)
	if err != nil {
		return fmt.Errorf("error trimming redis stream by age in append: %w", err)
	}

	maxAgeSeconds := int64(r.maxAge / time.Second)
	if maxAgeSeconds < 1 {
		maxAgeSeconds = 1
	}
	_, err = redigo.DoContext(connection, cctx, "EXPIRE", key, maxAgeSeconds)
	if err != nil {
		return fmt.Errorf("error setting inactive redis stream expiry in append: %w", err)
	}

	return nil
}

func replayCutoffID(now time.Time, maxAge time.Duration) string {
	return fmt.Sprintf("%d-0", now.Add(-maxAge).UnixMilli())
}

func (r *eventStreams) prepare(
	ctx context.Context,
	topicName string,
	lastEventID string,
) (*vibe.ReplaySubscription, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := r.pool.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in prepare: %w", err)
	}
	defer connection.Close()

	key := replayKey(topicName)
	firstID, err := readBoundaryID(cctx, connection, "XRANGE", key, "-", "+")
	if err != nil {
		return nil, fmt.Errorf("error reading first replay id in prepare: %w", err)
	}
	lastID, err := readBoundaryID(cctx, connection, "XREVRANGE", key, "+", "-")
	if err != nil {
		return nil, fmt.Errorf("error reading last replay id in prepare: %w", err)
	}

	if firstID == "" || lastID == "" {
		return &vibe.ReplaySubscription{
			AfterID:          initialStreamID,
			RequiresSnapshot: true,
		}, nil
	}

	normalizedLastEventID := strings.TrimSpace(lastEventID)
	if normalizedLastEventID == "" {
		return &vibe.ReplaySubscription{
			AfterID:          lastID,
			RequiresSnapshot: true,
		}, nil
	}

	valid, err := replayIDWithinBounds(normalizedLastEventID, firstID, lastID)
	if err != nil || !valid {
		return &vibe.ReplaySubscription{
			AfterID:          lastID,
			RequiresSnapshot: true,
		}, nil
	}

	return &vibe.ReplaySubscription{
		AfterID:          normalizedLastEventID,
		RequiresSnapshot: false,
	}, nil
}

func (r *eventStreams) subscribe(
	ctx context.Context,
	topicName string,
	afterID string,
	roomEvents bool,
) (*streamSubscription, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("error creating replay subscription in subscribe: redis pool is nil")
	}

	cursor := strings.TrimSpace(afterID)
	if cursor == "" {
		cursor = initialStreamID
	}
	_, err := parseStreamID(cursor)
	if err != nil {
		return nil, fmt.Errorf("error parsing replay cursor in subscribe: %w", err)
	}

	subscriptionContext, cancel := context.WithCancel(ctx)
	subscription := &streamSubscription{
		cancel:     cancel,
		done:       make(chan bool),
		key:        replayKey(topicName),
		messages:   make(chan []byte, roomEventSubscriptionBufferSize),
		pool:       r.pool,
		readCount:  r.maxEvents,
		roomEvents: roomEvents,
	}

	go subscription.listen(subscriptionContext, cursor)

	return subscription, nil
}

type streamSubscription struct {
	cancel     context.CancelFunc
	done       chan bool
	key        string
	messages   chan []byte
	once       sync.Once
	pool       *redigo.Pool
	readCount  int
	roomEvents bool
}

func (s *streamSubscription) Destroy() {
	s.once.Do(func() {
		s.cancel()
		<-s.done
	})
}

func (s *streamSubscription) Listen() chan []byte {
	return s.messages
}

func (s *streamSubscription) listen(ctx context.Context, cursor string) {
	defer close(s.done)
	defer close(s.messages)

	currentCursor := cursor
	for {
		if ctx.Err() != nil {
			return
		}

		messages, err := s.read(ctx, currentCursor)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("error reading replay stream for %s: %v", s.key, err)
			if !waitForReplayRetry(ctx) {
				return
			}
			continue
		}

		deliveries := messages
		if s.roomEvents {
			deliveries = compactReplayMessages(messages)
		}

		for _, message := range deliveries {
			select {
			case <-ctx.Done():
				return
			case s.messages <- message.Data:
				currentCursor = message.ID
			}
		}
	}
}

func (s *streamSubscription) read(
	ctx context.Context,
	cursor string,
) ([]streamMessage, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "readEventStream")
	defer span.End()

	connection, err := s.pool.GetContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in read: %w", err)
	}
	defer connection.Close()

	reply, err := redigo.DoContext(
		connection,
		ctx,
		"XREAD",
		"BLOCK",
		replayBlockMilliseconds,
		"COUNT",
		s.readCount,
		"STREAMS",
		s.key,
		cursor,
	)
	streams, err := redigo.Values(reply, err)
	if errors.Is(err, redigo.ErrNil) {
		return []streamMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing stream list in read: %w", err)
	}

	messages := []streamMessage{}
	for _, rawStream := range streams {
		stream, err := redigo.Values(rawStream, nil)
		if err != nil {
			return nil, fmt.Errorf("error parsing stream in read: %w", err)
		}
		if len(stream) != 2 {
			return nil, fmt.Errorf("error parsing stream in read: expected two values")
		}

		entries, err := redigo.Values(stream[1], nil)
		if err != nil {
			return nil, fmt.Errorf("error parsing stream entries in read: %w", err)
		}
		for _, rawEntry := range entries {
			entry, err := redigo.Values(rawEntry, nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing entry in read: %w", err)
			}
			if len(entry) != 2 {
				return nil, fmt.Errorf("error parsing entry in read: expected two values")
			}

			id, err := redigo.String(entry[0], nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing id in read: %w", err)
			}
			fields, err := redigo.StringMap(entry[1], nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing fields in read: %w", err)
			}
			data, ok := fields[replayEventField]
			if !ok {
				return nil, fmt.Errorf("error parsing fields in read: event is missing")
			}

			eventData := []byte(data)
			eventType := ""
			if s.roomEvents {
				var event vibe.RoomEvent
				err = json.Unmarshal(eventData, &event)
				if err != nil {
					return nil, fmt.Errorf("error unmarshaling room event in read: %w", err)
				}
				event.ID = id
				eventData, err = json.Marshal(event)
				if err != nil {
					return nil, fmt.Errorf("error marshaling room event in read: %w", err)
				}
				eventType = event.Type
			}

			messages = append(messages, streamMessage{ID: id, Data: eventData, Type: eventType})
		}
	}

	return messages, nil
}

type streamMessage struct {
	ID   string
	Data []byte
	Type string
}

func compactReplayMessages(messages []streamMessage) []streamMessage {
	keep := make([]bool, len(messages))
	latestOnlySeen := make(map[string]bool)
	queueSnapshotSeen := false
	usersSnapshotSeen := false

	for index := len(messages) - 1; index >= 0; index-- {
		messageType := messages[index].Type
		switch messageType {
		case vibe.QueueReordered:
			if !queueSnapshotSeen {
				keep[index] = true
				queueSnapshotSeen = true
			}
		case vibe.SongAdded, vibe.SongRemoved:
			keep[index] = !queueSnapshotSeen
		case vibe.UsersUpdate:
			if !usersSnapshotSeen {
				keep[index] = true
				usersSnapshotSeen = true
			}
		case vibe.UserJoined, vibe.UserLeft:
			keep[index] = !usersSnapshotSeen
		case vibe.PlaybackUpdate,
			vibe.SettingsUpdate,
			vibe.SkipVoteEvent,
			vibe.NewHost,
			vibe.GenerationUpdate:
			if !latestOnlySeen[messageType] {
				keep[index] = true
				latestOnlySeen[messageType] = true
			}
		default:
			keep[index] = true
		}
	}

	compacted := make([]streamMessage, 0, len(messages))
	for index, message := range messages {
		if keep[index] {
			compacted = append(compacted, message)
		}
	}

	return compacted
}

type streamID struct {
	Milliseconds int64
	Sequence     int64
}

func readBoundaryID(
	ctx context.Context,
	connection redigo.Conn,
	command string,
	key string,
	start string,
	end string,
) (string, error) {
	reply, err := redigo.DoContext(
		connection,
		ctx,
		command,
		key,
		start,
		end,
		"COUNT",
		1,
	)
	if err != nil {
		return "", fmt.Errorf("error reading redis stream boundary in readBoundaryID: %w", err)
	}

	entries, err := redigo.Values(reply, nil)
	if err != nil {
		return "", fmt.Errorf("error parsing redis stream boundary in readBoundaryID: %w", err)
	}
	if len(entries) == 0 {
		return "", nil
	}

	entry, err := redigo.Values(entries[0], nil)
	if err != nil {
		return "", fmt.Errorf("error parsing redis stream entry in readBoundaryID: %w", err)
	}
	if len(entry) == 0 {
		return "", fmt.Errorf("error parsing redis stream entry in readBoundaryID: entry is empty")
	}

	id, err := redigo.String(entry[0], nil)
	if err != nil {
		return "", fmt.Errorf("error parsing redis stream id in readBoundaryID: %w", err)
	}

	return id, nil
}

func replayIDWithinBounds(candidate string, first string, last string) (bool, error) {
	candidateID, err := parseStreamID(candidate)
	if err != nil {
		return false, fmt.Errorf("error parsing candidate stream id in replayIDWithinBounds: %w", err)
	}
	firstID, err := parseStreamID(first)
	if err != nil {
		return false, fmt.Errorf("error parsing first stream id in replayIDWithinBounds: %w", err)
	}
	lastID, err := parseStreamID(last)
	if err != nil {
		return false, fmt.Errorf("error parsing last stream id in replayIDWithinBounds: %w", err)
	}

	return compareStreamIDs(candidateID, firstID) >= 0 &&
		compareStreamIDs(candidateID, lastID) <= 0, nil
}

func parseStreamID(value string) (*streamID, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error invalid stream id %q", value)
	}

	milliseconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing stream milliseconds: %w", err)
	}
	sequence, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing stream sequence: %w", err)
	}
	if milliseconds < 0 || sequence < 0 {
		return nil, fmt.Errorf("error stream id must not be negative")
	}

	return &streamID{Milliseconds: milliseconds, Sequence: sequence}, nil
}

func compareStreamIDs(left *streamID, right *streamID) int {
	if left.Milliseconds < right.Milliseconds {
		return -1
	}
	if left.Milliseconds > right.Milliseconds {
		return 1
	}
	if left.Sequence < right.Sequence {
		return -1
	}
	if left.Sequence > right.Sequence {
		return 1
	}
	return 0
}

func waitForReplayRetry(ctx context.Context) bool {
	timer := time.NewTimer(replayRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func replayKey(topicName string) string {
	return fmt.Sprintf("%s:events:%s", redisPrefix, topicName)
}

func roomTopicName(roomID string) string {
	return fmt.Sprintf("room:%s", roomID)
}

func remoteTopicName(remoteID string) string {
	return fmt.Sprintf("remote:%s", remoteID)
}

const adminTopicName = "admin"
const replayEventField = "event"
const initialStreamID = "0-0"
const replayBlockMilliseconds = 3000
const replayRetryDelay = time.Second
const roomEventSubscriptionBufferSize = 32
