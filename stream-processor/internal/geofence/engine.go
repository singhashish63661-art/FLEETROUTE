package geofence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"gpsgo/stream-processor/internal/enrichment"
)

type Engine struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewEngine(pool *pgxpool.Pool, logger *zap.Logger) *Engine {
	return &Engine{pool: pool, logger: logger}
}

func (e *Engine) Check(ctx context.Context, rec *enrichment.EnrichedRecord) {
	// A fast ST_Contains query using PostGIS to check if the point intersects any geofence.
	// Normally we would cache the geometries, but PostGIS with a spatial index is extremely fast.
	
	rows, err := e.pool.Query(ctx, `
		SELECT id, name FROM geofences 
		WHERE tenant_id = $1 
		AND ST_Contains(geom, ST_SetSRID(ST_Point($2, $3), 4326))
	`, rec.TenantID, rec.Lng, rec.Lat)
	
	if err != nil {
		e.logger.Error("geofence check error", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var gid, name string
		if err := rows.Scan(&gid, &name); err == nil {
			// Trigger a fake alert / log for geofence violation
			e.logger.Debug("Device inside geofence", 
				zap.String("device_id", rec.DeviceID),
				zap.String("geofence", name))
		}
	}
}
