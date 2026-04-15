// Package redis_consumer subscribes to Redis Pub/Sub channels and broadcasts
// messages to the WebSocket Hub for real-time fan-out to connected clients.
package redis_consumer

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkgdb "gpsgo/pkg/db"
	"gpsgo/websocket-service/internal/hub"
)

// Consumer subscribes to Redis Pub/Sub and forwards messages to the Hub.
type Consumer struct {
	rdb    *redis.Client
	hub    *hub.Hub
	logger *zap.Logger
}

// New creates a Consumer.
func New(rdb *redis.Client, h *hub.Hub, logger *zap.Logger) *Consumer {
	return &Consumer{rdb: rdb, hub: h, logger: logger}
}

// Subscribe starts the Redis Pub/Sub subscription loop.
// It subscribes to all tenant position channels using a pattern.
func (c *Consumer) Subscribe(ctx context.Context) {
	// Subscribe to wildcard pattern — receives all tenant device updates
	// Pattern: pubsub:device:* and pubsub:alerts:*
	pubsub := c.rdb.PSubscribe(ctx, "pubsub:device:*", "pubsub:alerts:*")
	defer pubsub.Close()

	c.logger.Info("redis pubsub subscribed to device and alert channels")

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			c.handleMessage(msg)
		}
	}
}

func (c *Consumer) handleMessage(msg *redis.Message) {
	// Extract tenant ID from channel name
	// Channel: pubsub:device:{tenantID}
	var tenantID, deviceID string

	channel := msg.Channel
	switch {
	case len(channel) > len(pkgdb.KeyPubSubDevice("")):
		tenantID = channel[len(pkgdb.KeyPubSubDevice("")):]
	default:
		return
	}

	// Decode to extract device_id for subscription filtering
	var payload map[string]any
	if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
		if d, ok := payload["device_id"].(string); ok {
			deviceID = d
		}
	}

	c.hub.Broadcast(tenantID, deviceID, []byte(msg.Payload))
}
