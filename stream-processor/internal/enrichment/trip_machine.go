package enrichment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	StateIdle   = "IDLE"
	StateActive = "ACTIVE"
	StateEnded  = "ENDED"
)

type TripMachine struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	logger *zap.Logger
}

func NewTripMachine(pool *pgxpool.Pool, rdb *redis.Client, logger *zap.Logger) *TripMachine {
	return &TripMachine{pool: pool, rdb: rdb, logger: logger}
}

func (m *TripMachine) Process(ctx context.Context, rec *EnrichedRecord) {
	stateKey := fmt.Sprintf("gpsgo:tripstate:%s", rec.DeviceID)
	tripIdKey := fmt.Sprintf("gpsgo:tripstate:%s:tripid", rec.DeviceID)

	state, err := m.rdb.Get(ctx, stateKey).Result()
	if err == redis.Nil {
		state = StateIdle
	}

	if state == StateIdle && rec.Speed > 5 {
		// Transition to ACTIVE
		var newTripID string
		err := m.pool.QueryRow(ctx, `
			INSERT INTO trips (tenant_id, device_id, started_at, start_lat, start_lng)
			VALUES ($1, $2, $3, $4, $5) RETURNING id
		`, rec.TenantID, rec.DeviceID, rec.Timestamp, rec.Lat, rec.Lng).Scan(&newTripID)

		if err != nil {
			m.logger.Error("failed to create trip", zap.Error(err))
			return
		}

		m.rdb.Set(ctx, stateKey, StateActive, 0)
		m.rdb.Set(ctx, tripIdKey, newTripID, 0)
		m.logger.Info("started trip", zap.String("trip_id", newTripID))

	} else if state == StateActive && rec.Speed < 2 {
		tripID, _ := m.rdb.Get(ctx, tripIdKey).Result()
		if tripID != "" {
			_, err := m.pool.Exec(ctx, `
				UPDATE trips 
				SET ended_at = $1, end_lat = $2, end_lng = $3
				WHERE id = $4
			`, rec.Timestamp, rec.Lat, rec.Lng, tripID)
			
			if err != nil {
				m.logger.Error("failed to end trip", zap.Error(err))
			} else {
				m.logger.Info("ended trip", zap.String("trip_id", tripID))
			}
		}
		
		m.rdb.Set(ctx, stateKey, StateIdle, 0)
		m.rdb.Del(ctx, tripIdKey)
	}
}
