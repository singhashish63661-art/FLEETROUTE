// Package teltonika implements the Teltonika AVL protocol handler supporting
// Codec 8, Codec 8E, and Codec 16 as used by FMB/FMC/FMT device families.
//
// Protocol reference: https://wiki.teltonika-gps.com/view/Codec
package teltonika

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"gpsgo/pkg/protocol"
)

const (
	Codec8  = 0x08
	Codec8E = 0x8E
	Codec16 = 0x10

	preamble = uint32(0x00000000)
)

// Handler implements protocol.Handler for the Teltonika AVL protocol.
type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Name() string { return "teltonika" }

// Detect returns true if the header looks like a Teltonika IMEI handshake.
// Teltonika connections start with: 2-byte IMEI length (0x000F = 15) followed by ASCII IMEI digits.
func (h *Handler) Detect(header []byte) bool {
	if len(header) < 2 {
		return false
	}
	imeiLen := binary.BigEndian.Uint16(header[0:2])
	return imeiLen >= 15 && imeiLen <= 17
}

// Authenticate reads and validates the IMEI handshake, responds with 0x01 (accepted).
func (h *Handler) Authenticate(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck

	// Read 2-byte IMEI length
	var imeiLen uint16
	if err := binary.Read(conn, binary.BigEndian, &imeiLen); err != nil {
		return "", fmt.Errorf("teltonika: read imei length: %w", err)
	}
	if imeiLen < 15 || imeiLen > 17 {
		return "", fmt.Errorf("teltonika: invalid imei length %d", imeiLen)
	}

	// Read IMEI bytes
	imeiBytes := make([]byte, imeiLen)
	if _, err := io.ReadFull(conn, imeiBytes); err != nil {
		return "", fmt.Errorf("teltonika: read imei: %w", err)
	}
	imei := string(imeiBytes)

	// Validate: IMEI must be all digits
	for _, c := range imei {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("teltonika: invalid imei chars: %q", imei)
		}
	}

	// Send ACK: 0x01 = accepted
	if _, err := conn.Write([]byte{0x01}); err != nil {
		return "", fmt.Errorf("teltonika: send ack: %w", err)
	}

	conn.SetDeadline(time.Time{}) //nolint:errcheck
	return imei, nil
}

// ParsePacket reads one full AVL data packet from the connection and returns parsed records.
//
// Teltonika data packet structure (Codec 8/8E):
//
//	[0x00000000] [data_length: 4B] [codec_id: 1B] [num_data: 1B]
//	[AVL_records: ...] [num_data: 1B] [CRC16-IBM: 4B]
func (h *Handler) ParsePacket(conn net.Conn) ([]protocol.ParsedRecord, []byte, error) {
	conn.SetDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	// Read preamble (4 bytes, must be 0x00000000)
	var pre uint32
	if err := binary.Read(conn, binary.BigEndian, &pre); err != nil {
		return nil, nil, fmt.Errorf("teltonika: read preamble: %w", err)
	}
	if pre != preamble {
		return nil, nil, fmt.Errorf("teltonika: bad preamble 0x%08X", pre)
	}

	// Read data length
	var dataLen uint32
	if err := binary.Read(conn, binary.BigEndian, &dataLen); err != nil {
		return nil, nil, fmt.Errorf("teltonika: read data length: %w", err)
	}
	if dataLen > 65536 {
		return nil, nil, fmt.Errorf("teltonika: suspicious data length %d", dataLen)
	}

	// Read full payload
	payload := make([]byte, dataLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, nil, fmt.Errorf("teltonika: read payload: %w", err)
	}

	// Read CRC (4 bytes, lower 2 bytes are CRC16-IBM)
	var crcRaw uint32
	if err := binary.Read(conn, binary.BigEndian, &crcRaw); err != nil {
		return nil, nil, fmt.Errorf("teltonika: read crc: %w", err)
	}
	expectedCRC := uint16(crcRaw & 0xFFFF)
	actualCRC := CRC16IBM(payload)
	if expectedCRC != actualCRC {
		return nil, payload, fmt.Errorf("teltonika: CRC mismatch: expected 0x%04X got 0x%04X", expectedCRC, actualCRC)
	}

	// Build the full raw packet for DLQ retention
	rawPacket := make([]byte, 4+4+dataLen+4)
	binary.BigEndian.PutUint32(rawPacket[0:], preamble)
	binary.BigEndian.PutUint32(rawPacket[4:], dataLen)
	copy(rawPacket[8:], payload)
	binary.BigEndian.PutUint32(rawPacket[8+dataLen:], crcRaw)

	// Decode codec
	if len(payload) < 2 {
		return nil, rawPacket, fmt.Errorf("teltonika: payload too short")
	}
	codecID := payload[0]
	numRecords := int(payload[1])

	var records []protocol.ParsedRecord
	var err error

	switch codecID {
	case Codec8:
		records, err = parseCodec8(payload[2:], numRecords)
	case Codec8E:
		records, err = parseCodec8E(payload[2:], numRecords)
	case Codec16:
		records, err = parseCodec16(payload[2:], numRecords)
	default:
		return nil, rawPacket, fmt.Errorf("teltonika: unsupported codec 0x%02X", codecID)
	}
	if err != nil {
		return nil, rawPacket, fmt.Errorf("teltonika codec 0x%02X parse: %w", codecID, err)
	}

	for i := range records {
		records[i].RawCodec = codecID
	}

	return records, rawPacket, nil
}

// BuildACK returns the 4-byte ACK with the count of records received.
func (h *Handler) BuildACK(packet []byte) []byte {
	// Count from payload[1] (num_data_1 byte)
	var count uint32
	if len(packet) >= 9 {
		count = uint32(packet[9]) // preamble(4)+len(4)+codec(1) = offset 9 for num_data
	}
	ack := make([]byte, 4)
	binary.BigEndian.PutUint32(ack, count)
	return ack
}

// ── Codec 8 Parser ─────────────────────────────────────────────────────────────

func parseCodec8(data []byte, numRecords int) ([]protocol.ParsedRecord, error) {
	records := make([]protocol.ParsedRecord, 0, numRecords)
	offset := 0

	for i := 0; i < numRecords; i++ {
		rec, n, err := parseAVLRecord8(data, offset, false)
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", i, err)
		}
		records = append(records, rec)
		offset += n
	}
	return records, nil
}

// ── Codec 8E Parser ────────────────────────────────────────────────────────────

func parseCodec8E(data []byte, numRecords int) ([]protocol.ParsedRecord, error) {
	records := make([]protocol.ParsedRecord, 0, numRecords)
	offset := 0

	for i := 0; i < numRecords; i++ {
		rec, n, err := parseAVLRecord8(data, offset, true) // extended = true
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", i, err)
		}
		records = append(records, rec)
		offset += n
	}
	return records, nil
}

// ── Codec 16 Parser ───────────────────────────────────────────────────────────

func parseCodec16(data []byte, numRecords int) ([]protocol.ParsedRecord, error) {
	// Codec 16 has an additional 2-byte Generation Time field before each record
	records := make([]protocol.ParsedRecord, 0, numRecords)
	offset := 0

	for i := 0; i < numRecords; i++ {
		if offset+2 > len(data) {
			return nil, fmt.Errorf("codec16 record %d: truncated generation time", i)
		}
		// Skip 2-byte generation time (record generation time, not timestamp)
		offset += 2
		rec, n, err := parseAVLRecord8(data, offset, false)
		if err != nil {
			return nil, fmt.Errorf("codec16 record %d: %w", i, err)
		}
		records = append(records, rec)
		offset += n
	}
	return records, nil
}

// ── Core AVL Record Parser ────────────────────────────────────────────────────

// parseAVLRecord8 parses a single AVL data record for Codec 8 / 8E.
// extended=true enables 2-byte IO element IDs (Codec 8E).
// Returns the record and the number of bytes consumed.
func parseAVLRecord8(data []byte, offset int, extended bool) (protocol.ParsedRecord, int, error) {
	start := offset
	rec := protocol.ParsedRecord{IOData: make(map[int]int64)}

	need := func(n int) error {
		if offset+n > len(data) {
			return fmt.Errorf("need %d bytes at offset %d, have %d", n, offset, len(data))
		}
		return nil
	}

	// ── Timestamp (8 bytes, Unix ms) ─────────────────────────────────────────
	if err := need(8); err != nil {
		return rec, 0, fmt.Errorf("timestamp: %w", err)
	}
	rec.Timestamp = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// ── Priority (1 byte) ────────────────────────────────────────────────────
	if err := need(1); err != nil {
		return rec, 0, fmt.Errorf("priority: %w", err)
	}
	rec.Priority = data[offset]
	offset++

	// ── GPS Element (15 bytes) ────────────────────────────────────────────────
	// Longitude, Latitude: 4 bytes each (×10^7)
	// Altitude: 2 bytes
	// Angle: 2 bytes
	// Satellites: 1 byte
	// Speed: 2 bytes
	if err := need(15); err != nil {
		return rec, 0, fmt.Errorf("gps element: %w", err)
	}
	lngRaw := int32(binary.BigEndian.Uint32(data[offset:]))
	rec.Lng = float64(lngRaw) / 10_000_000.0
	offset += 4

	latRaw := int32(binary.BigEndian.Uint32(data[offset:]))
	rec.Lat = float64(latRaw) / 10_000_000.0
	offset += 4

	rec.Altitude = int32(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	rec.Heading = binary.BigEndian.Uint16(data[offset:])
	offset += 2

	rec.Satellites = data[offset]
	offset++

	rec.Speed = binary.BigEndian.Uint16(data[offset:])
	offset += 2

	// GNSS fix validity: satellites > 0 and coordinates non-zero
	rec.Valid = rec.Satellites > 0 && (rec.Lat != 0 || rec.Lng != 0)

	// ── IO Element Header ─────────────────────────────────────────────────────
	if extended {
		// Codec 8E: 2-byte event IO ID, 2-byte total IO count
		if err := need(4); err != nil {
			return rec, 0, fmt.Errorf("8E io header: %w", err)
		}
		offset += 2 // event IO ID (skip — we parse all)
		offset += 2 // total IO element count
	} else {
		// Codec 8: 1-byte event IO ID, 1-byte total IO count
		if err := need(2); err != nil {
			return rec, 0, fmt.Errorf("8 io header: %w", err)
		}
		offset += 1 // event IO ID
		offset += 1 // total count
	}

	// ── IO Elements by value width ────────────────────────────────────────────
	// Parse groups: 1B, 2B, 4B, 8B (and for 8E also X-byte / variable)
	for _, valueBytes := range []int{1, 2, 4, 8} {
		var count int
		if extended {
			if err := need(2); err != nil {
				return rec, 0, fmt.Errorf("8E io count (%dB): %w", valueBytes, err)
			}
			count = int(binary.BigEndian.Uint16(data[offset:]))
			offset += 2
		} else {
			if err := need(1); err != nil {
				return rec, 0, fmt.Errorf("8 io count (%dB): %w", valueBytes, err)
			}
			count = int(data[offset])
			offset++
		}

		for j := 0; j < count; j++ {
			// IO element ID
			var ioID int
			if extended {
				if err := need(2); err != nil {
					return rec, 0, fmt.Errorf("8E io id: %w", err)
				}
				ioID = int(binary.BigEndian.Uint16(data[offset:]))
				offset += 2
			} else {
				if err := need(1); err != nil {
					return rec, 0, fmt.Errorf("8 io id: %w", err)
				}
				ioID = int(data[offset])
				offset++
			}

			// IO element value
			if err := need(valueBytes); err != nil {
				return rec, 0, fmt.Errorf("io value (id=%d, len=%d): %w", ioID, valueBytes, err)
			}
			var val int64
			switch valueBytes {
			case 1:
				val = int64(data[offset])
			case 2:
				val = int64(binary.BigEndian.Uint16(data[offset:]))
			case 4:
				val = int64(binary.BigEndian.Uint32(data[offset:]))
			case 8:
				val = int64(binary.BigEndian.Uint64(data[offset:]))
			}
			rec.IOData[ioID] = val
			offset += valueBytes
		}
	}

	// Codec 8E has an additional X-byte (variable width) group
	if extended {
		if err := need(2); err != nil {
			return rec, 0, fmt.Errorf("8E xbyte count: %w", err)
		}
		xCount := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2
		for j := 0; j < xCount; j++ {
			if err := need(4); err != nil {
				return rec, 0, fmt.Errorf("8E xbyte element header: %w", err)
			}
			ioID := int(binary.BigEndian.Uint16(data[offset:]))
			offset += 2
			xLen := int(binary.BigEndian.Uint16(data[offset:]))
			offset += 2
			// Store raw hex of variable-length data as a byte slice encoded int64
			_ = ioID
			_ = xLen
			if err := need(xLen); err != nil {
				return rec, 0, fmt.Errorf("8E xbyte value: %w", err)
			}
			// For now, store the first 8 bytes as int64 (variable data logged separately)
			var val int64
			for k := 0; k < xLen && k < 8; k++ {
				val = (val << 8) | int64(data[offset+k])
			}
			rec.IOData[ioID] = val
			offset += xLen
		}
	}

	return rec, offset - start, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// TimestampToTime converts a Teltonika Unix-ms timestamp to time.Time.
func TimestampToTime(unixMs int64) time.Time {
	return time.UnixMilli(unixMs).UTC()
}

// IMEIFromHex decodes a hex-encoded IMEI string (used in testing).
func IMEIFromHex(h string) (string, error) {
	b, err := hex.DecodeString(strings.ReplaceAll(h, " ", ""))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
