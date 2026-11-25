package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetOrClaimMetadata atomically reads a verified source result or leases one
// provider lookup across all backend pods. Error reports discard cached results.
func (c *Client) GetOrClaimMetadata(ctx context.Context, provider, sourceID string, force bool) (*vibe.ProviderMetadataLookup, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetOrClaimMetadata")
	defer span.End()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting metadata redis connection: %w", err)
	}
	defer connection.Close()
	key := c.getKeyWithPrefix("metadata:" + provider + ":" + sourceID)
	token := uuid.NewString()
	values, err := redis.Values(redis.DoContext(connection, cctx, "EVAL", claimMetadataScript, 2, key, key+":claim", force, token))
	if err != nil {
		return nil, fmt.Errorf("error claiming provider metadata: %w", err)
	}
	var body string
	var claimed string
	_, err = redis.Scan(values, &body, &claimed)
	if err != nil {
		return nil, fmt.Errorf("error reading metadata claim: %w", err)
	}
	lookup := vibe.ProviderMetadataLookup{Token: claimed}
	if body != "" {
		err = json.Unmarshal([]byte(body), &lookup.Metadata)
		if err != nil {
			return nil, fmt.Errorf("error decoding cached metadata: %w", err)
		}
	}
	return &lookup, nil
}

func (c *Client) CompleteMetadata(ctx context.Context, provider, sourceID, token string, metadata vibe.ProviderMetadata, lifetime time.Duration) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CompleteMetadata")
	defer span.End()
	body, err := json.Marshal(metadata)
	if err != nil {
		return false, fmt.Errorf("error encoding verified metadata: %w", err)
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return false, fmt.Errorf("error getting metadata redis connection: %w", err)
	}
	defer connection.Close()
	key := c.getKeyWithPrefix("metadata:" + provider + ":" + sourceID)
	completed, err := redis.Bool(redis.DoContext(connection, cctx, "EVAL", completeMetadataScript, 2, key, key+":claim", token, body, lifetime.Milliseconds()))
	if err != nil {
		return false, fmt.Errorf("error completing metadata claim: %w", err)
	}
	return completed, nil
}

func (c *Client) ReleaseMetadata(ctx context.Context, provider, sourceID, token string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReleaseMetadata")
	defer span.End()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting metadata redis connection: %w", err)
	}
	defer connection.Close()
	_, err = redis.DoContext(connection, cctx, "DELEX", c.getKeyWithPrefix("metadata:"+provider+":"+sourceID+":claim"), "IFEQ", token)
	if err != nil {
		return fmt.Errorf("error releasing metadata claim: %w", err)
	}
	return nil
}

const claimMetadataScript = `
if ARGV[1] == '1' then redis.call('DEL', KEYS[1]) end
local cached = redis.call('GET', KEYS[1])
if cached then return {cached, ''} end
if redis.call('SET', KEYS[2], ARGV[2], 'NX', 'PX', 60000) then
 return {'', ARGV[2]}
end
return {'', ''}
`

const completeMetadataScript = `
if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
redis.call('DEL', KEYS[2])
return 1
`
