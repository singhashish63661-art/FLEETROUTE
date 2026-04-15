// Package protocol defines the contract every GPS device protocol handler must satisfy.
package protocol

import "net"

// Handler is the interface every protocol implementation must satisfy.
// Implementations live in gpsgo/protocols/* and are linked into ingestion-service.
type Handler interface {
	// Name returns the protocol identifier (e.g. "teltonika", "gt06").
	Name() string

	// Detect returns true if the supplied header bytes indicate this protocol.
	// header is at most 16 bytes read non-destructively from the connection buffer.
	Detect(header []byte) bool

	// Authenticate performs the device authentication handshake on conn.
	// It MUST send the appropriate ACK and return the authenticated device IMEI/ID.
	Authenticate(conn net.Conn) (deviceID string, err error)

	// ParsePacket parses a single raw packet read from conn into one or more
	// AVL records. Store-and-forward devices may return multiple records per packet.
	// The returned raw []byte is the full packet (for dead-letter retention on error).
	ParsePacket(conn net.Conn) (records []ParsedRecord, raw []byte, err error)

	// BuildACK constructs the byte-level acknowledgement for a successfully
	// parsed data packet. packet is the raw bytes of the packet being ACK'd.
	BuildACK(packet []byte) []byte
}

// ParsedRecord is a raw parsed record before enrichment. It carries the AVL
// data plus wire-level metadata needed for idempotency and dead-lettering.
type ParsedRecord struct {
	DeviceID  string
	Timestamp int64            // Unix ms (device clock)
	Lat       float64
	Lng       float64
	Altitude  int32
	Speed     uint16
	Heading   uint16
	Satellites uint8
	Valid      bool
	Priority   uint8
	IOData    map[int]int64
	RawCodec  uint8
}
