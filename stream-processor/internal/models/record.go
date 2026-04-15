package models

import "time"

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
