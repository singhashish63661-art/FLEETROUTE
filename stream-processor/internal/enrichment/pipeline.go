package enrichment

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gpsgo/pkg/protocol"
)

// EnrichedRecord is the post-enrichment record with named telemetry fields.
type EnrichedRecord struct {
	DeviceID   string    `json:"device_id"`
	TenantID   string    `json:"tenant_id"`
	MessageID  string    `json:"message_id"`
	Timestamp  time.Time `json:"timestamp"`
	ReceivedAt time.Time `json:"received_at"`

	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Altitude   int32   `json:"altitude"`
	Speed      uint16  `json:"speed"`
	Heading    uint16  `json:"heading"`
	Satellites uint8   `json:"satellites"`
	Valid      bool    `json:"valid"`
	Priority   uint8   `json:"priority"`

	Ignition        bool    `json:"ignition"`
	Movement        bool    `json:"movement"`
	ExternalVoltage float64 `json:"external_voltage_v"`
	BatteryVoltage  float64 `json:"battery_voltage_v"`
	BatteryLevel    int     `json:"battery_level_pct"`
	GNSSStatus      int     `json:"gnss_status"`
	GSMSignal       int     `json:"gsm_signal"`
	Speed_CAN       int     `json:"can_speed_kmh"`
	EngineRPM       int     `json:"engine_rpm"`
	FuelLevel       int     `json:"fuel_level_pct"`
	Temperature1    float64 `json:"temperature_1_c"`
	UserID          string  `json:"user_id"`
	SOSEvent        bool    `json:"sos_event"`

	IOData   map[int]int64 `json:"io_data"`
	RawCodec uint8         `json:"raw_codec"`
}

// Enricher is a single step in the enrichment pipeline.
type Enricher func(ctx context.Context, rec *EnrichedRecord)

// Pipeline holds an ordered list of enrichers.
type Pipeline struct {
	pool      *pgxpool.Pool
	enrichers []Enricher
	logger    *zap.Logger

	tripMachine *TripMachine
	geoEngine   *GeofenceEngine
	alertsEval  *AlertEvaluator
	notifMgr    *NotificationManager
}

// NewPipeline constructs the enrichment pipeline.
func NewPipeline(pool *pgxpool.Pool, rdb *redis.Client, logger *zap.Logger) *Pipeline {
	p := &Pipeline{
		pool:        pool,
		logger:      logger,
		tripMachine: NewTripMachine(pool, rdb, logger),
		geoEngine:   NewGeofenceEngine(pool, logger),
		alertsEval:  NewAlertEvaluator(pool, logger),
		notifMgr:    NewNotificationManager(logger),
	}
	p.enrichers = []Enricher{
		p.enrichIOFields,
		p.enrichTripState,
		p.enrichGeofence,
		p.enrichAlerts,
	}
	return p
}

// Process converts a raw parsed record into an enriched record.
func (p *Pipeline) Process(ctx context.Context, raw protocol.ParsedRecord) *EnrichedRecord {
	tenantID := ""
	if p.pool != nil && raw.DeviceID != "" {
		err := p.pool.QueryRow(ctx,
			`SELECT tenant_id::text FROM devices WHERE id = $1::uuid AND deleted_at IS NULL`,
			raw.DeviceID,
		).Scan(&tenantID)
		if err != nil {
			p.logger.Warn("tenant lookup for device failed",
				zap.String("device_id", raw.DeviceID), zap.Error(err))
		}
	}

	rec := &EnrichedRecord{
		DeviceID:   raw.DeviceID,
		TenantID:   tenantID,
		Timestamp:  time.UnixMilli(raw.Timestamp).UTC(),
		ReceivedAt: time.Now().UTC(),
		Lat:        raw.Lat,
		Lng:        raw.Lng,
		Altitude:   raw.Altitude,
		Speed:      raw.Speed,
		Heading:    raw.Heading,
		Satellites: raw.Satellites,
		Valid:       raw.Valid,
		Priority:   raw.Priority,
		IOData:     raw.IOData,
		RawCodec:   raw.RawCodec,
	}

	for _, fn := range p.enrichers {
		fn(ctx, rec)
	}
	return rec
}

func (p *Pipeline) enrichIOFields(_ context.Context, rec *EnrichedRecord) {
	io := rec.IOData

	if io == nil {
		return
	}

	// Ignition — IO 239 (Teltonika) or IO 1 (Digital Input 1)
	if v, ok := io[239]; ok {
		rec.Ignition = v == 1
	} else if v, ok := io[1]; ok {
		rec.Ignition = v == 1
	}

	if v, ok := io[240]; ok {
		rec.Movement = v == 1
	}
	if v, ok := io[66]; ok {
		rec.ExternalVoltage = float64(v) / 1000.0
	}
	if v, ok := io[67]; ok {
		rec.BatteryVoltage = float64(v) / 1000.0
	}
	if v, ok := io[113]; ok {
		rec.BatteryLevel = int(v)
	}
	if v, ok := io[69]; ok {
		rec.GNSSStatus = int(v)
	}
	if v, ok := io[21]; ok {
		rec.GSMSignal = int(v)
	}
	if v, ok := io[263]; ok {
		rec.EngineRPM = int(v)
	}
	if v, ok := io[327]; ok {
		rec.EngineRPM = int(v)
	}
	if v, ok := io[274]; ok {
		rec.FuelLevel = int(v)
	}
	if v, ok := io[72]; ok {
		rec.Temperature1 = float64(v) / 10.0
	}
	if v, ok := io[238]; ok {
		rec.UserID = fmt.Sprintf("%X", v)
	}
	if v, ok := io[236]; ok {
		rec.SOSEvent = v == 1
	}
	if v, ok := io[320]; ok {
		rec.Speed_CAN = int(v)
	}
}

func (p *Pipeline) enrichTripState(ctx context.Context, rec *EnrichedRecord) {
	p.tripMachine.Process(ctx, rec)
}

func (p *Pipeline) enrichGeofence(ctx context.Context, rec *EnrichedRecord) {
	p.geoEngine.Check(ctx, rec)
}

func (p *Pipeline) enrichAlerts(ctx context.Context, rec *EnrichedRecord) {
	p.alertsEval.Evaluate(ctx, rec)
}


