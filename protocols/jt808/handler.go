// Package jt808 implements the JT/T 808 GPS tracker protocol handler.
// JT808 is the Chinese national standard for vehicle tracking (JT/T 808-2019).
// Transport: TCP. Encoding: Binary with BCD fields.
//
// Frame structure:
//   0x7E [header: 12B] [body: N] [checksum: 1B] 0x7E
//   With escape: 0x7D 0x02 → 0x7E, 0x7D 0x01 → 0x7D
package jt808

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
	frameByte  = 0x7E
	escapeByte = 0x7D

	// Message IDs
	msgIDTerminalRegister    = 0x0100
	msgIDTerminalAuth        = 0x0102
	msgIDLocationReport      = 0x0200
	msgIDLocationBatch       = 0x0704
	msgIDHeartbeat           = 0x0002
	msgIDPlatformAck         = 0x8001
	msgIDLocationAck         = 0x8002
	msgIDRegisterAck         = 0x8100
)

// Handler implements protocol.Handler for JT808 devices.
type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Name() string { return "jt808" }

// Detect identifies JT808 by its 0x7E start byte.
func (h *Handler) Detect(header []byte) bool {
	return len(header) > 0 && header[0] == frameByte
}

// Authenticate performs JT808 device registration + authentication.
// Sequence: Terminal Register (0x0100) → Platform ACK (0x8100) → Terminal Auth (0x0102) → Platform ACK (0x8001)
func (h *Handler) Authenticate(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	// Read terminal register message
	pkt, err := readFrame(conn)
	if err != nil {
		return "", fmt.Errorf("jt808 auth read register: %w", err)
	}

	var deviceID string
	if pkt.msgID == msgIDTerminalRegister {
		// Extract phone number (BCD, 6 bytes at offset 10 in body)
		deviceID = bcdPhoneFromHeader(pkt.phoneNum)

		// Send register ACK (0x8100): seq + result(0=success) + auth_code
		authCode := "GPSGO_AUTH"
		respBody := make([]byte, 3+len(authCode))
		binary.BigEndian.PutUint16(respBody[0:], pkt.seqNum)
		respBody[2] = 0x00 // success
		copy(respBody[3:], []byte(authCode))
		if err := writeFrame(conn, msgIDRegisterAck, pkt.phoneNum, pkt.seqNum, respBody); err != nil {
			return "", fmt.Errorf("jt808 send register ack: %w", err)
		}
	} else if pkt.msgID == msgIDTerminalAuth {
		// Some devices skip register and go straight to auth
		deviceID = bcdPhoneFromHeader(pkt.phoneNum)
	} else {
		return "", fmt.Errorf("jt808: unexpected msg 0x%04X during auth", pkt.msgID)
	}

	// If it was a register, read the following auth message
	if pkt.msgID == msgIDTerminalRegister {
		authPkt, err := readFrame(conn)
		if err != nil {
			return "", fmt.Errorf("jt808 auth read auth msg: %w", err)
		}
		if authPkt.msgID != msgIDTerminalAuth {
			return "", fmt.Errorf("jt808: expected auth msg 0x0102, got 0x%04X", authPkt.msgID)
		}
	}

	// Send platform ACK for auth
	ackBody := make([]byte, 5)
	binary.BigEndian.PutUint16(ackBody[0:], pkt.seqNum)
	binary.BigEndian.PutUint16(ackBody[2:], msgIDTerminalAuth)
	ackBody[4] = 0x00 // success
	if err := writeFrame(conn, msgIDPlatformAck, pkt.phoneNum, pkt.seqNum+1, ackBody); err != nil {
		return "", fmt.Errorf("jt808 send auth ack: %w", err)
	}

	return deviceID, nil
}

// ParsePacket reads and parses one JT808 data packet.
func (h *Handler) ParsePacket(conn net.Conn) ([]protocol.ParsedRecord, []byte, error) {
	conn.SetDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	pkt, err := readFrame(conn)
	if err != nil {
		return nil, nil, err
	}

	switch pkt.msgID {
	case msgIDLocationReport:
		rec, err := parseLocation(pkt.body)
		if err != nil {
			return nil, pkt.raw, err
		}
		// ACK
		ackBody := make([]byte, 5)
		binary.BigEndian.PutUint16(ackBody[0:], pkt.seqNum)
		binary.BigEndian.PutUint16(ackBody[2:], msgIDLocationReport)
		ackBody[4] = 0x00
		writeFrame(conn, msgIDLocationAck, pkt.phoneNum, pkt.seqNum+1, ackBody) //nolint:errcheck
		return []protocol.ParsedRecord{rec}, pkt.raw, nil

	case msgIDLocationBatch:
		records, err := parseLocationBatch(pkt.body)
		if err != nil {
			return nil, pkt.raw, err
		}
		ackBody := make([]byte, 5)
		binary.BigEndian.PutUint16(ackBody[0:], pkt.seqNum)
		binary.BigEndian.PutUint16(ackBody[2:], msgIDLocationBatch)
		ackBody[4] = 0x00
		writeFrame(conn, msgIDLocationAck, pkt.phoneNum, pkt.seqNum+1, ackBody) //nolint:errcheck
		return records, pkt.raw, nil

	case msgIDHeartbeat:
		ackBody := make([]byte, 5)
		binary.BigEndian.PutUint16(ackBody[0:], pkt.seqNum)
		binary.BigEndian.PutUint16(ackBody[2:], msgIDHeartbeat)
		ackBody[4] = 0x00
		writeFrame(conn, msgIDPlatformAck, pkt.phoneNum, pkt.seqNum+1, ackBody) //nolint:errcheck
	}

	return nil, pkt.raw, nil
}

func (h *Handler) BuildACK(_ []byte) []byte { return nil }

// ── JT808 Frame Types ─────────────────────────────────────────────────────────

type jt808Frame struct {
	msgID    uint16
	phoneNum [6]byte
	seqNum   uint16
	body     []byte
	raw      []byte
}

// readFrame reads one complete JT808 frame, handling escape sequences.
func readFrame(conn net.Conn) (*jt808Frame, error) {
	// Find start 0x7E
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, fmt.Errorf("jt808 read start: %w", err)
		}
		if buf[0] == frameByte {
			break
		}
	}

	// Read until end 0x7E, collecting escaped bytes
	var escaped []byte
	for {
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, fmt.Errorf("jt808 read body: %w", err)
		}
		if buf[0] == frameByte {
			break
		}
		escaped = append(escaped, buf[0])
	}

	// Unescape: 0x7D 0x01 → 0x7D, 0x7D 0x02 → 0x7E
	data := unescape(escaped)

	// Validate checksum (XOR of all bytes except last)
	if len(data) < 13 {
		return nil, fmt.Errorf("jt808 frame too short: %d", len(data))
	}
	checksum := data[len(data)-1]
	var xor byte
	for _, b := range data[:len(data)-1] {
		xor ^= b
	}
	if xor != checksum {
		return nil, fmt.Errorf("jt808 checksum mismatch: got 0x%02X want 0x%02X", xor, checksum)
	}

	// Parse header (12 bytes):
	// msg_id(2) + attr(2) + phone(6) + seq(2)
	frame := &jt808Frame{
		msgID:  binary.BigEndian.Uint16(data[0:2]),
		seqNum: binary.BigEndian.Uint16(data[10:12]),
		body:   data[12 : len(data)-1],
	}
	copy(frame.phoneNum[:], data[4:10])

	// Reconstruct raw including frame bytes
	frame.raw = append([]byte{frameByte}, append(escape(data), frameByte)...)

	return frame, nil
}

func writeFrame(conn net.Conn, msgID uint16, phoneNum [6]byte, seqNum uint16, body []byte) error {
	// Build header
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:], msgID)
	binary.BigEndian.PutUint16(header[2:], uint16(len(body))) // msg attr: body length
	copy(header[4:10], phoneNum[:])
	binary.BigEndian.PutUint16(header[10:], seqNum)

	data := append(header, body...)

	// Compute checksum
	var xor byte
	for _, b := range data {
		xor ^= b
	}
	data = append(data, xor)

	// Escape and frame
	framed := append([]byte{frameByte}, append(escape(data), frameByte)...)
	_, err := conn.Write(framed)
	return err
}

// parseLocation parses a JT808 0x0200 location report body.
func parseLocation(body []byte) (protocol.ParsedRecord, error) {
	rec := protocol.ParsedRecord{IOData: make(map[int]int64)}
	if len(body) < 28 {
		return rec, fmt.Errorf("jt808 location body too short: %d", len(body))
	}

	// Alarm(4) + Status(4) + Lat(4) + Lng(4) + Altitude(2) + Speed(2) + Direction(2) + Time(6 BCD)
	alarmFlags := binary.BigEndian.Uint32(body[0:4])
	statusFlags := binary.BigEndian.Uint32(body[4:8])

	latRaw := binary.BigEndian.Uint32(body[8:12])
	lngRaw := binary.BigEndian.Uint32(body[12:16])

	lat := float64(latRaw) / 1e6
	lng := float64(lngRaw) / 1e6

	// Status bits: bit 2 = S/N, bit 3 = E/W
	if (statusFlags>>2)&1 == 0 { // 0=South
		lat = -lat
	}
	if (statusFlags>>3)&1 == 0 { // 0=West
		lng = -lng
	}

	altitude := binary.BigEndian.Uint16(body[16:18])
	speed := binary.BigEndian.Uint16(body[18:20])     // 1/10 km/h
	direction := binary.BigEndian.Uint16(body[20:22]) // 0-359

	// BCD time: YYMMDDHHmmss
	ts := parseBCDTime(body[22:28])

	rec.Timestamp = ts.UnixMilli()
	rec.Lat = lat
	rec.Lng = lng
	rec.Altitude = int32(altitude)
	rec.Speed = uint16(float64(speed) / 10.0)
	rec.Heading = direction
	rec.Valid = (statusFlags>>1)&1 == 1 // bit 1 = GPS fix

	// Map alarm flags to IO elements
	if alarmFlags&(1<<0) != 0 { // emergency alarm / SOS
		rec.IOData[236] = 1
	}
	if alarmFlags&(1<<3) != 0 { // movement alarm
		rec.IOData[240] = 1
	}
	// Ignition from status bit 5
	if (statusFlags>>5)&1 == 1 {
		rec.IOData[239] = 1
	}

	_ = alarmFlags
	return rec, nil
}

func parseLocationBatch(body []byte) ([]protocol.ParsedRecord, error) {
	if len(body) < 4 {
		return nil, fmt.Errorf("jt808 batch too short")
	}
	count := binary.BigEndian.Uint16(body[0:2])
	// body[2] = type (0=normal, 1=blind)
	offset := 3
	records := make([]protocol.ParsedRecord, 0, count)

	for i := 0; i < int(count); i++ {
		if offset+2 > len(body) {
			break
		}
		itemLen := int(binary.BigEndian.Uint16(body[offset:]))
		offset += 2
		if offset+itemLen > len(body) {
			break
		}
		rec, err := parseLocation(body[offset : offset+itemLen])
		if err == nil {
			records = append(records, rec)
		}
		offset += itemLen
	}
	return records, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func unescape(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == escapeByte && i+1 < len(data) {
			switch data[i+1] {
			case 0x01:
				out = append(out, escapeByte)
			case 0x02:
				out = append(out, frameByte)
			}
			i++
		} else {
			out = append(out, data[i])
		}
	}
	return out
}

func escape(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for _, b := range data {
		switch b {
		case frameByte:
			out = append(out, escapeByte, 0x02)
		case escapeByte:
			out = append(out, escapeByte, 0x01)
		default:
			out = append(out, b)
		}
	}
	return out
}

func bcdPhoneFromHeader(bcd [6]byte) string {
	var sb strings.Builder
	for _, b := range bcd {
		hi := (b >> 4) & 0x0F
		lo := b & 0x0F
		if hi != 0x0F {
			sb.WriteByte('0' + hi)
		}
		if lo != 0x0F {
			sb.WriteByte('0' + lo)
		}
	}
	return sb.String()
}

func parseBCDTime(bcd []byte) time.Time {
	if len(bcd) < 6 {
		return time.Now().UTC()
	}
	year := 2000 + int(bcdToDec(bcd[0]))
	month := int(bcdToDec(bcd[1]))
	day := int(bcdToDec(bcd[2]))
	hour := int(bcdToDec(bcd[3]))
	min := int(bcdToDec(bcd[4]))
	sec := int(bcdToDec(bcd[5]))
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
}

func bcdToDec(b byte) byte {
	return (b>>4)*10 + (b & 0x0F)
}
