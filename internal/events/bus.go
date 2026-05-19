// Package events implements the asynchronous event bus.
//
// Backed by Redis Streams: a single `onefacture.events` stream carries every
// domain event. Webhook delivery and PA status polling consume from the same
// log and use independent consumer groups so they can be scaled and recovered
// independently.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/yawo/onefacture/internal/config"
)

// Event is the canonical domain event published on the bus.
type Event struct {
	Type           string         `json:"type"`             // e.g. "invoice.submitted"
	OrganizationID string         `json:"organization_id"`
	InvoiceID      string         `json:"invoice_id,omitempty"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Payload        map[string]any `json:"payload,omitempty"`
}

// Bus is the messaging facade.
type Bus struct {
	rdb    *redis.Client
	stream string
}

// New connects to Redis and returns a ready-to-use bus.
func New(ctx context.Context, cfg config.RedisConfig) (*Bus, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Bus{rdb: rdb, stream: cfg.StreamKey}, nil
}

// Client returns the underlying Redis client for components that need it
// (e.g. rate limiter, locks).
func (b *Bus) Client() *redis.Client { return b.rdb }

// Publish appends an event to the stream.
func (b *Bus) Publish(ctx context.Context, ev Event) error {
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return b.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: b.stream,
		Values: map[string]any{"data": data},
	}).Err()
}

// Subscribe creates a consumer group (idempotent) and yields events to fn.
// The caller blocks here; cancel the context to stop.
func (b *Bus) Subscribe(ctx context.Context, group, consumer string, fn func(context.Context, Event) error) error {
	// Ensure group exists.
	if err := b.rdb.XGroupCreateMkStream(ctx, b.stream, group, "0").Err(); err != nil {
		if !isGroupExists(err) {
			return fmt.Errorf("create group: %w", err)
		}
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res, err := b.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{b.stream, ">"},
			Count:    20,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			return fmt.Errorf("xreadgroup: %w", err)
		}
		for _, s := range res {
			for _, m := range s.Messages {
				raw, ok := m.Values["data"].(string)
				if !ok {
					_ = b.rdb.XAck(ctx, b.stream, group, m.ID).Err()
					continue
				}
				var ev Event
				if err := json.Unmarshal([]byte(raw), &ev); err != nil {
					_ = b.rdb.XAck(ctx, b.stream, group, m.ID).Err()
					continue
				}
				if err := fn(ctx, ev); err != nil {
					// Leave un-acked for redelivery; production code would track retries.
					continue
				}
				_ = b.rdb.XAck(ctx, b.stream, group, m.ID).Err()
			}
		}
	}
}

func isGroupExists(err error) bool {
	return err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists"
}

// Close shuts down the Redis client.
func (b *Bus) Close() {
	if b.rdb != nil {
		_ = b.rdb.Close()
	}
}
