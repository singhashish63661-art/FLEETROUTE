package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkgdb "gpsgo/pkg/db"
	"gpsgo/stream-processor/internal/enrichment"
)

// RedisWriter updates the live device state in Redis and publishes to Pub/Sub.
type RedisWriter struct {
	rdb    *redis.Client
	logger *zap.Logger
}

// NewRedisWriter creates a RedisWriter.
func NewRedisWriter(rdb *redis.Client, logger *zap.Logger) *RedisWriter {
	return &RedisWriter{rdb: rdb, logger: logger}
}

// UpdateLive updates the device:live:{tenant}:{device} key with the current position snapshot.
// TTL is 60 seconds; if the device goes silent, the key expires automatically.
func (w *RedisWriter) UpdateLive(ctx context.Context, rec *enrichment.EnrichedRecord) error {
	key := pkgdb.KeyDeviceLive(rec.TenantID, rec.DeviceID)

	snapshot := map[string]any{
		"device_id":        rec.DeviceID,
		"tenant_id":        rec.TenantID,
		"timestamp":        rec.Timestamp.UnixMilli(),
		"lat":              rec.Lat,
		"lng":              rec.Lng,
		"altitude":         rec.Altitude,
		"speed":            rec.Speed,
		"heading":          rec.Heading,
		"satellites":       rec.Satellites,
		"valid":            rec.Valid,
		"ignition":         rec.Ignition,
		"movement":         rec.Movement,
		"external_voltage": rec.ExternalVoltage,
		"battery_level":    rec.BatteryLevel,
		"gsm_signal":       rec.GSMSignal,
		"sos_event":        rec.SOSEvent,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal live snapshot: %w", err)
	}

	return w.rdb.Set(ctx, key, data, 60*time.Second).Err()
}

// PublishLive publishes a position update to Redis Pub/Sub for WebSocket fan-out.
// Channel is namespaced by tenant so WS subscribers only receive their data.
func (w *RedisWriter) PublishLive(ctx context.Context, rec *enrichment.EnrichedRecord) error {
	channel := pkgdb.KeyPubSubDevice(rec.TenantID)

	msg := map[string]any{
		"type":      "position",
		"device_id": rec.DeviceID,
		"tenant_id": rec.TenantID,
		"timestamp": rec.Timestamp.UnixMilli(),
		"lat":       rec.Lat,
		"lng":       rec.Lng,
		"speed":     rec.Speed,
		"heading":   rec.Heading,
		"ignition":  rec.Ignition,
		"sos_event": rec.SOSEvent,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal pubsub message: %w", err)
	}

	return w.rdb.Publish(ctx, channel, data).Err()
}

// jsonMarshal is a local alias to avoid import issues.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
