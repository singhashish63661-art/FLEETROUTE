// Package avl defines the core data types for AVL (Automatic Vehicle Location) records
// produced by GPS tracking devices across all supported protocols.
package avl

import "time"

// Priority defines the AVL record priority level as specified by Teltonika.
type Priority uint8

const (
	PriorityLow    Priority = 0
	PriorityHigh   Priority = 1
	PriorityPanic  Priority = 2
)

// AVLRecord is the normalised, protocol-agnostic representation of a single
// position + telemetry snapshot from any supported device.
type AVLRecord struct {
	// Identity
	DeviceID  string    `json:"device_id"`
	TenantID  string    `json:"tenant_id"`
	MessageID string    `json:"message_id"` // idempotency key

	// Timing
	Timestamp  time.Time `json:"timestamp"`   // device timestamp (UTC)
	ReceivedAt time.Time `json:"received_at"` // server ingestion time

	// Position
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Altitude int32   `json:"altitude"`  // metres
	Speed    uint16  `json:"speed"`     // km/h
	Heading  uint16  `json:"heading"`   // degrees (0–359)

	// GNSS Quality
	Satellites uint8   `json:"satellites"`
	Accuracy   float32 `json:"accuracy"` // HDOP or metres
	Valid       bool    `json:"valid"`    // GNSS fix valid

	// Metadata
	Priority Priority `json:"priority"`
	RawCodec uint8    `json:"raw_codec"` // 8, 8E, 16, etc.

	// Telemetry — IO element ID → raw int64 value.
	// Interpretation is done by the enrichment pipeline using IORegistry.
	IOData map[int]int64 `json:"io_data"`
}

// GPSElement holds the raw GPS fields before normalisation. Used internally
// by protocol parsers before constructing an AVLRecord.
type GPSElement struct {
	Longitude  float64
	Latitude   float64
	Altitude   int16
	Angle      uint16
	Satellites uint8
	Speed      uint16
}

// IOElement holds a single IO data element key-value pair.
type IOElement struct {
	ID    int
	Value int64
}
