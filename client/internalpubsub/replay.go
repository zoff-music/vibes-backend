package internalpubsub

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

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/vibe"
)

type Replay struct {
	Pool      *redis.Pool
	MaxEvents int
	MaxAge    time.Duration
}

func (r *Replay) Append(ctx context.Context, topicName string, data []byte) error {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := r.Pool.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in Append: %w", err)
	}
	defer connection.Close()

	key := replayKey(topicName)
	_, err = redis.DoContext(
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
	_, err = redis.DoContext(
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
	_, err = redis.DoContext(connection, cctx, "EXPIRE", key, maxAgeSeconds)
	if err != nil {
		return fmt.Errorf("error setting inactive redis stream expiry in Append: %w", err)
	}

	return nil
}

func replayCutoffID(now time.Time, maxAge time.Duration) string {
	return fmt.Sprintf("%d-0", now.Add(-maxAge).UnixMilli())
}

func (r *Replay) Prepare(
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

func (r *Replay) Subscribe(
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
		messages:  make(chan []byte, subscriptionBufferSize),
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
	pool      *redis.Pool
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

	reply, err := redis.DoContext(
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
	if errors.Is(err, redis.ErrNil) {
		return []StreamMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading redis stream in read: %w", err)
	}

	streams, err := redis.Values(reply, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing stream list in read: %w", err)
	}

	messages := []StreamMessage{}
	for _, rawStream := range streams {
		stream, err := redis.Values(rawStream, nil)
		if err != nil {
			return nil, fmt.Errorf("error parsing stream in read: %w", err)
		}
		if len(stream) != 2 {
			return nil, fmt.Errorf("error parsing stream in read: expected two values")
		}

		entries, err := redis.Values(stream[1], nil)
		if err != nil {
			return nil, fmt.Errorf("error parsing stream entries in read: %w", err)
		}
		for _, rawEntry := range entries {
			entry, err := redis.Values(rawEntry, nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing entry in read: %w", err)
			}
			if len(entry) != 2 {
				return nil, fmt.Errorf("error parsing entry in read: expected two values")
			}

			id, err := redis.String(entry[0], nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing id in read: %w", err)
			}
			fields, err := redis.StringMap(entry[1], nil)
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
	connection redis.Conn,
	command string,
	key string,
	start string,
	end string,
) (string, error) {
	reply, err := redis.DoContext(
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

	entries, err := redis.Values(reply, nil)
	if err != nil {
		return "", fmt.Errorf("error parsing redis stream boundary in readBoundaryID: %w", err)
	}
	if len(entries) == 0 {
		return "", nil
	}

	entry, err := redis.Values(entries[0], nil)
	if err != nil {
		return "", fmt.Errorf("error parsing redis stream entry in readBoundaryID: %w", err)
	}
	if len(entry) == 0 {
		return "", fmt.Errorf("error parsing redis stream entry in readBoundaryID: entry is empty")
	}

	id, err := redis.String(entry[0], nil)
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
	return fmt.Sprintf("%s:%s", replayKeyPrefix, topicName)
}

const replayKeyPrefix = "vibes:events"
const replayEventField = "event"
const initialStreamID = "0-0"
const replayBlockMilliseconds = 3000
const replayRetryDelay = time.Second
