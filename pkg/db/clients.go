// Package db provides database client setup for TimescaleDB (pgx) and Redis.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ── TimescaleDB (PostgreSQL + pgx) ────────────────────────────────────────────

// NewPool creates a pgxpool connection pool for TimescaleDB.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool parse config: %w", err)
	}
	cfg.MaxConns = 50
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}
	return pool, nil
}

// ── Redis ─────────────────────────────────────────────────────────────────────

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewRedis creates a go-redis client and verifies connectivity.
func NewRedis(ctx context.Context, cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
		PoolTimeout:  4 * time.Second,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

// ── Redis Key Patterns ────────────────────────────────────────────────────────
// All keys are namespaced by tenant to enforce isolation.

// KeyDeviceLive returns the Redis key for a device's current live position snapshot.
// TTL: 60 seconds (refreshed on every AVL record).
func KeyDeviceLive(tenantID, deviceID string) string {
	return fmt.Sprintf("device:live:%s:%s", tenantID, deviceID)
}

// KeyDeviceSession returns the Redis key for a device's TCP connection metadata.
func KeyDeviceSession(deviceID string) string {
	return fmt.Sprintf("device:session:%s", deviceID)
}

// KeyGeofenceIndex returns the Redis key for a tenant's geofence spatial index.
func KeyGeofenceIndex(tenantID string) string {
	return fmt.Sprintf("geofence:index:%s", tenantID)
}

// KeyTripActive returns the Redis key for a device's active trip state machine.
func KeyTripActive(deviceID string) string {
	return fmt.Sprintf("trip:active:%s", deviceID)
}

// KeyPubSubDevice returns the Redis Pub/Sub channel for real-time device updates.
func KeyPubSubDevice(tenantID string) string {
	return fmt.Sprintf("pubsub:device:%s", tenantID)
}

// KeyPubSubAlerts returns the Redis Pub/Sub channel for real-time alerts.
func KeyPubSubAlerts(tenantID string) string {
	return fmt.Sprintf("pubsub:alerts:%s", tenantID)
}

// KeyRuleCooldown returns the Redis key for a rule's per-device cooldown tracking.
func KeyRuleCooldown(ruleID, deviceID string) string {
	return fmt.Sprintf("rule:cooldown:%s:%s", ruleID, deviceID)
}
