package enrichment

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type AlertEvaluator struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewAlertEvaluator(pool *pgxpool.Pool, logger *zap.Logger) *AlertEvaluator {
	return &AlertEvaluator{pool: pool, logger: logger}
}

func (e *AlertEvaluator) Evaluate(ctx context.Context, rec *EnrichedRecord) {
	if rec.Speed > 120 {
		e.logger.Warn("Overspeed alert triggered", zap.String("device", rec.DeviceID))
		
		_, err := e.pool.Exec(ctx, `
			INSERT INTO alerts (tenant_id, device_id, alert_type, severity, message, lat, lng, speed, triggered_at)
			VALUES ($1, $2, 'overspeed', 'warning', 'Vehicle exceeded 120km/h', $3, $4, $5, $6)
		`, rec.TenantID, rec.DeviceID, rec.Lat, rec.Lng, rec.Speed, rec.Timestamp)

		if err != nil {
			e.logger.Error("failed to insert alert", zap.Error(err))
		}
	}

	if rec.SOSEvent {
		e.logger.Error("SOS alert triggered", zap.String("device", rec.DeviceID))
		_, err := e.pool.Exec(ctx, `
			INSERT INTO alerts (tenant_id, device_id, alert_type, severity, message, lat, lng, speed, triggered_at)
			VALUES ($1, $2, 'sos', 'critical', 'SOS Button Pressed!', $3, $4, $5, $6)
		`, rec.TenantID, rec.DeviceID, rec.Lat, rec.Lng, rec.Speed, rec.Timestamp)

		if err != nil {
			e.logger.Error("failed to insert sos alert", zap.Error(err))
		}
	}
}
