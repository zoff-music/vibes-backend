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
	"github.com/zoff-music/vibes-backend/vibe"
)

type roomEventReplay struct {
	Pool      *redigo.Pool
	MaxEvents int
	MaxAge    time.Duration
}

func (c *Client) NotifyRoomUpdate(
	ctx context.Context,
	roomID string,
	event vibe.RoomEvent,
) error {
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
	for _, event := range events {
		err := c.NotifyRoomUpdate(ctx, roomID, event)
		if err != nil {
			return fmt.Errorf("error notifying room event batch in NotifyRoomUpdates: %w", err)
		}
	}

	return nil
}

func (c *Client) appendRoomEvent(ctx context.Context, topicName string, data []byte) error {
	if c.replay == nil {
		return fmt.Errorf("error appending room event: replay is not initialized")
	}

	err := c.replay.Append(ctx, topicName, data)
	if err != nil {
		return fmt.Errorf("error appending room event: %w", err)
	}

	return nil
}

func (c *Client) PrepareReplay(
	ctx context.Context,
	topicName string,
	lastEventID string,
) (*vibe.ReplaySubscription, error) {
	if c.replay == nil {
		return nil, fmt.Errorf("error preparing room event replay: replay is not initialized")
	}

	replay, err := c.replay.Prepare(ctx, topicName, lastEventID)
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
	if c.replay == nil {
		return nil, fmt.Errorf("error subscribing to room events: replay is not initialized")
	}

	subscription, err := c.replay.Subscribe(ctx, topicName, afterID)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to room events: %w", err)
	}

	return &vibe.SubscriptionContainer{Subscription: subscription}, nil
}

func (c *Client) Subscribe(topicName string) (*vibe.SubscriptionContainer, error) {
	replay, err := c.PrepareReplay(context.Background(), topicName, "")
	if err != nil {
		return nil, fmt.Errorf("error preparing room events in Subscribe: %w", err)
	}

	container, err := c.SubscribeFrom(context.Background(), topicName, replay.AfterID)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to room events in Subscribe: %w", err)
	}

	return container, nil
}

func (r *roomEventReplay) Append(ctx context.Context, topicName string, data []byte) error {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := r.Pool.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in Append: %w", err)
	}
	defer connection.Close()

	key := replayKey(topicName)
	_, err = redigo.DoContext(
		connection,
		cctx,
		"XADD",
		key,
		"MAXLEN",
		r.MaxEvents,
		"*",
		replayEventField,
		data,
	)
	if err != nil {
		return fmt.Errorf("error appending redis stream event in Append: %w", err)
	}

	cutoffID := replayCutoffID(time.Now(), r.MaxAge)
	_, err = redigo.DoContext(
		connection,
		cctx,
		"XTRIM",
		key,
		"MINID",
		cutoffID,
	)
	if err != nil {
		return fmt.Errorf("error trimming redis stream by age in Append: %w", err)
	}

	maxAgeSeconds := int64(r.MaxAge / time.Second)
	if maxAgeSeconds < 1 {
		maxAgeSeconds = 1
	}
	_, err = redigo.DoContext(connection, cctx, "EXPIRE", key, maxAgeSeconds)
	if err != nil {
		return fmt.Errorf("error setting inactive redis stream expiry in Append: %w", err)
	}

	return nil
}

func replayCutoffID(now time.Time, maxAge time.Duration) string {
	return fmt.Sprintf("%d-0", now.Add(-maxAge).UnixMilli())
}

func (r *roomEventReplay) Prepare(
	ctx context.Context,
	topicName string,
	lastEventID string,
) (*vibe.ReplaySubscription, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := r.Pool.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in Prepare: %w", err)
	}
	defer connection.Close()

	key := replayKey(topicName)
	firstID, err := readBoundaryID(cctx, connection, "XRANGE", key, "-", "+")
	if err != nil {
		return nil, fmt.Errorf("error reading first replay id in Prepare: %w", err)
	}
	lastID, err := readBoundaryID(cctx, connection, "XREVRANGE", key, "+", "-")
	if err != nil {
		return nil, fmt.Errorf("error reading last replay id in Prepare: %w", err)
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

func (r *roomEventReplay) Subscribe(
	ctx context.Context,
	topicName string,
	afterID string,
) (*StreamSubscription, error) {
	if r.Pool == nil {
		return nil, fmt.Errorf("error creating replay subscription in Subscribe: redis pool is nil")
	}

	cursor := strings.TrimSpace(afterID)
	if cursor == "" {
		cursor = initialStreamID
	}
	_, err := parseStreamID(cursor)
	if err != nil {
		return nil, fmt.Errorf("error parsing replay cursor in Subscribe: %w", err)
	}

	subscriptionContext, cancel := context.WithCancel(ctx)
	subscription := &StreamSubscription{
		cancel:    cancel,
		done:      make(chan struct{}),
		key:       replayKey(topicName),
		messages:  make(chan []byte, roomEventSubscriptionBufferSize),
		pool:      r.Pool,
		readCount: r.MaxEvents,
	}

	go subscription.listen(subscriptionContext, cursor)

	return subscription, nil
}

type StreamSubscription struct {
	cancel    context.CancelFunc
	done      chan struct{}
	key       string
	messages  chan []byte
	once      sync.Once
	pool      *redigo.Pool
	readCount int
}

func (s *StreamSubscription) Destroy() {
	s.once.Do(func() {
		s.cancel()
		<-s.done
	})
}

func (s *StreamSubscription) Listen() chan []byte {
	return s.messages
}

func (s *StreamSubscription) listen(ctx context.Context, cursor string) {
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

		for _, message := range compactReplayMessages(messages) {
			select {
			case <-ctx.Done():
				return
			case s.messages <- message.Data:
				currentCursor = message.ID
			}
		}
	}
}

func (s *StreamSubscription) read(
	ctx context.Context,
	cursor string,
) ([]StreamMessage, error) {
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
	if errors.Is(err, redigo.ErrNil) {
		return []StreamMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading redis stream in read: %w", err)
	}

	streams, err := redigo.Values(reply, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing stream list in read: %w", err)
	}

	messages := []StreamMessage{}
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

			var event vibe.RoomEvent
			err = json.Unmarshal([]byte(data), &event)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling room event in read: %w", err)
			}
			event.ID = id
			eventData, err := json.Marshal(event)
			if err != nil {
				return nil, fmt.Errorf("error marshaling room event in read: %w", err)
			}

			messages = append(messages, StreamMessage{ID: id, Data: eventData, Type: event.Type})
		}
	}

	return messages, nil
}

type StreamMessage struct {
	ID   string
	Data []byte
	Type string
}

func compactReplayMessages(messages []StreamMessage) []StreamMessage {
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

	compacted := make([]StreamMessage, 0, len(messages))
	for index, message := range messages {
		if keep[index] {
			compacted = append(compacted, message)
		}
	}

	return compacted
}

type StreamID struct {
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

func parseStreamID(value string) (*StreamID, error) {
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

	return &StreamID{Milliseconds: milliseconds, Sequence: sequence}, nil
}

func compareStreamIDs(left *StreamID, right *StreamID) int {
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

const replayEventField = "event"
const initialStreamID = "0-0"
const replayBlockMilliseconds = 3000
const replayRetryDelay = time.Second
const roomEventSubscriptionBufferSize = 32
