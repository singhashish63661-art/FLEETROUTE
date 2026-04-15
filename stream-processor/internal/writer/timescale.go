package writer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"gpsgo/stream-processor/internal/enrichment"
)

// TimescaleWriter writes enriched AVL records to the avl_records hypertable.
type TimescaleWriter struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewTimescaleWriter creates a TimescaleWriter.
func NewTimescaleWriter(pool *pgxpool.Pool, logger *zap.Logger) *TimescaleWriter {
	return &TimescaleWriter{pool: pool, logger: logger}
}

// Write inserts a single enriched record.
func (w *TimescaleWriter) Write(ctx context.Context, rec *enrichment.EnrichedRecord) error {
	ioJSON, _ := json.Marshal(rec.IOData)

	sql := `
		INSERT INTO avl_records (
			device_id, tenant_id, timestamp, received_at,
			lat, lng, altitude, speed, heading, satellites, valid,
			priority, raw_codec,
			ignition, movement,
			external_voltage, battery_voltage, battery_level,
			gnss_status, gsm_signal,
			engine_rpm, fuel_level, temperature_1,
			user_id, sos_event,
			io_data
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10, $11,
			$12, $13,
			$14, $15,
			$16, $17, $18,
			$19, $20,
			$21, $22, $23,
			$24, $25,
			$26
		) ON CONFLICT DO NOTHING`

	_, err := w.pool.Exec(ctx, sql,
		rec.DeviceID, rec.TenantID, rec.Timestamp, rec.ReceivedAt,
		rec.Lat, rec.Lng, rec.Altitude, rec.Speed, rec.Heading, rec.Satellites, rec.Valid,
		rec.Priority, rec.RawCodec,
		rec.Ignition, rec.Movement,
		rec.ExternalVoltage, rec.BatteryVoltage, rec.BatteryLevel,
		rec.GNSSStatus, rec.GSMSignal,
		rec.EngineRPM, rec.FuelLevel, rec.Temperature1,
		rec.UserID, rec.SOSEvent,
		ioJSON,
	)
	if err != nil {
		return fmt.Errorf("avl_records insert: %w", err)
	}

	_, err = w.pool.Exec(ctx,
		`UPDATE devices SET last_seen_at = $1, updated_at = NOW() WHERE id = $2::uuid`,
		rec.Timestamp, rec.DeviceID,
	)
	if err != nil {
		w.logger.Debug("devices last_seen update", zap.String("device_id", rec.DeviceID), zap.Error(err))
	}
	return nil
}
