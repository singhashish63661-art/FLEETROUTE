// Package gt06 implements the protocol handler for GT06/GT06N GPS trackers.
// Transport: TCP. Protocol: Binary with IMEI authentication.
//
// Packet format:
//   Start:  0x78 0x78 (2B)
//   Length: 1B
//   Proto:  1B (message type)
//   Info:   N bytes
//   Serial: 2B
//   CRC:    2B (CRC-16/CCITT-FALSE)
//   Stop:   0x0D 0x0A (2B)
package gt06

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"gpsgo/pkg/protocol"
)

const (
	startByte1 = 0x78
	startByte2 = 0x78

	protoLogin    = 0x01
	protoGPSSingle = 0x12
	protoGPSBatch  = 0x28
	protoHeartbeat = 0x23
	protoAlarm     = 0x26
)

// Handler implements protocol.Handler for GT06/GT06N devices.
type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Name() string { return "gt06" }

// Detect identifies GT06 by its 0x78 0x78 start bytes.
func (h *Handler) Detect(header []byte) bool {
	if len(header) < 2 {
		return false
	}
	return header[0] == startByte1 && header[1] == startByte2
}

// Authenticate handles the GT06 login packet (proto 0x01).
// The login packet contains the 8-byte IMEI in BCD format.
func (h *Handler) Authenticate(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	pkt, err := readPacket(conn)
	if err != nil {
		return "", fmt.Errorf("gt06 auth read packet: %w", err)
	}
	if pkt.proto != protoLogin {
		return "", fmt.Errorf("gt06: expected login packet (0x01), got 0x%02X", pkt.proto)
	}

	// Decode BCD IMEI from 8 bytes
	if len(pkt.info) < 8 {
		return "", fmt.Errorf("gt06: login info too short: %d bytes", len(pkt.info))
	}
	imei := decodeBCDIMEI(pkt.info[:8])

	// Send response: start+start + length(0x05) + proto(0x01) + serial(2B) + CRC(2B) + stop
	resp := buildResponse(protoLogin, pkt.serial, []byte{})
	if _, err := conn.Write(resp); err != nil {
		return "", fmt.Errorf("gt06: send login ack: %w", err)
	}

	return imei, nil
}

// ParsePacket reads and parses one GT06 data packet.
func (h *Handler) ParsePacket(conn net.Conn) ([]protocol.ParsedRecord, []byte, error) {
	conn.SetDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	pkt, err := readPacket(conn)
	if err != nil {
		return nil, nil, err
	}

	var records []protocol.ParsedRecord

	switch pkt.proto {
	case protoGPSSingle:
		rec, err := parseGPS(pkt.info)
		if err != nil {
			return nil, pkt.raw, err
		}
		records = append(records, rec)

		// Send ACK
		ack := buildResponse(protoGPSSingle, pkt.serial, []byte{})
		conn.Write(ack) //nolint:errcheck

	case protoHeartbeat:
		// Send heartbeat response
		ack := buildResponse(protoHeartbeat, pkt.serial, []byte{})
		conn.Write(ack) //nolint:errcheck

	case protoAlarm:
		rec, err := parseAlarm(pkt.info)
		if err != nil {
			return nil, pkt.raw, err
		}
		records = append(records, rec)
		ack := buildResponse(protoAlarm, pkt.serial, []byte{})
		conn.Write(ack) //nolint:errcheck
	}

	return records, pkt.raw, nil
}

func (h *Handler) BuildACK(packet []byte) []byte {
	return []byte{} // ACK built inline during ParsePacket
}

// ── Internal Types ────────────────────────────────────────────────────────────

type gt06Packet struct {
	proto  byte
	info   []byte
	serial uint16
	raw    []byte
}

func readPacket(conn net.Conn) (*gt06Packet, error) {
	// Read start bytes
	start := make([]byte, 2)
	if _, err := io.ReadFull(conn, start); err != nil {
		return nil, fmt.Errorf("gt06 start: %w", err)
	}
	if start[0] != startByte1 || start[1] != startByte2 {
		return nil, fmt.Errorf("gt06 bad start: %02X %02X", start[0], start[1])
	}

	// Length
	var length byte
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("gt06 length: %w", err)
	}

	// Read remaining: proto + info + serial(2) + crc(2) + stop(2)
	remaining := make([]byte, int(length)+4) // +4 = serial(2)+crc(2)+stop(2) - already counted differently
	// Actual: proto(1) + info(len-5) + serial(2) + crc(2) = length bytes, then stop 0x0D0A
	body := make([]byte, int(length))
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, fmt.Errorf("gt06 body: %w", err)
	}
	stop := make([]byte, 2)
	if _, err := io.ReadFull(conn, stop); err != nil {
		return nil, fmt.Errorf("gt06 stop: %w", err)
	}

	// Validate CRC
	crcData := body[:len(body)-4]
	expectedCRC := binary.BigEndian.Uint16(body[len(body)-4:len(body)-2])
	actualCRC := crc16CCITT(crcData)
	if expectedCRC != actualCRC {
		return nil, fmt.Errorf("gt06 CRC mismatch: expected 0x%04X got 0x%04X", expectedCRC, actualCRC)
	}

	raw := append([]byte{startByte1, startByte2, length}, append(body, stop...)...)

	_ = remaining
	return &gt06Packet{
		proto:  body[0],
		info:   body[1 : len(body)-4],
		serial: binary.BigEndian.Uint16(body[len(body)-4 : len(body)-2]),
		raw:    raw,
	}, nil
}

func parseGPS(info []byte) (protocol.ParsedRecord, error) {
	if len(info) < 12 {
		return protocol.ParsedRecord{}, fmt.Errorf("gt06 GPS info too short: %d", len(info))
	}
	// Date: year(1)+month(1)+day(1)+hour(1)+min(1)+sec(1)
	year := 2000 + int(info[0])
	month := int(info[1])
	day := int(info[2])
	hour := int(info[3])
	min := int(info[4])
	sec := int(info[5])
	ts := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)

	// GPS: len(1) + satellites(4bit) | fix(4bit) | lat(4B) | lng(4B) | speed(1) | heading(2) | ...
	gpsInfoLen := int(info[6])
	if len(info) < 7+gpsInfoLen {
		return protocol.ParsedRecord{}, fmt.Errorf("gt06 GPS data truncated")
	}
	satAndFix := info[7]
	satellites := (satAndFix >> 4) & 0x0F
	fix := satAndFix & 0x0F

	latRaw := binary.BigEndian.Uint32(info[8:])
	lngRaw := binary.BigEndian.Uint32(info[12:])
	lat := float64(latRaw) / 1800000.0
	lng := float64(lngRaw) / 1800000.0

	speed := info[16]
	courseRaw := binary.BigEndian.Uint16(info[17:])
	course := courseRaw & 0x03FF // lower 10 bits
	// Bit 10: lng E/W, bit 11: lat N/S
	if (courseRaw>>10)&1 == 0 {
		lng = -lng
	}
	if (courseRaw>>11)&1 == 0 {
		lat = -lat
	}

	return protocol.ParsedRecord{
		Timestamp:  ts.UnixMilli(),
		Lat:        lat,
		Lng:        lng,
		Speed:      uint16(speed),
		Heading:    course,
		Satellites: satellites,
		Valid:       fix > 0,
		RawCodec:   0,
		IOData:     make(map[int]int64),
	}, nil
}

func parseAlarm(info []byte) (protocol.ParsedRecord, error) {
	rec, err := parseGPS(info)
	if err != nil {
		return rec, err
	}
	// Alarm type is at a specific offset; store in IOData as SOS
	rec.IOData[236] = 1 // SOS event
	return rec, nil
}

func buildResponse(proto byte, serial uint16, data []byte) []byte {
	length := byte(5 + len(data)) // proto(1) + data + serial(2) + crc(2) = 5+len
	body := make([]byte, 0, int(length))
	body = append(body, proto)
	body = append(body, data...)
	body = append(body, byte(serial>>8), byte(serial))
	crc := crc16CCITT(body)
	body = append(body, byte(crc>>8), byte(crc))

	pkt := []byte{startByte1, startByte2, length}
	pkt = append(pkt, body...)
	pkt = append(pkt, 0x0D, 0x0A)
	return pkt
}

func decodeBCDIMEI(b []byte) string {
	var sb strings.Builder
	for _, v := range b {
		hi := (v >> 4) & 0x0F
		lo := v & 0x0F
		if hi <= 9 {
			sb.WriteByte('0' + hi)
		}
		if lo <= 9 {
			sb.WriteByte('0' + lo)
		}
	}
	return sb.String()
}

func crc16CCITT(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
