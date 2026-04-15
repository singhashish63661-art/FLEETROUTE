// Package ais140 implements the AIS140 VLTD (Vehicle Location Tracking Device)
// protocol mandated by India's Ministry of Road Transport for commercial vehicles.
//
// Reference: AIS 140:2016 - Intelligent Transport System (ITS)
// Packet format uses '$' header with comma-separated fields, terminated by '*' and checksum.
package ais140

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"gpsgo/pkg/protocol"
)

// Handler implements protocol.Handler for AIS140/VLTD devices.
type Handler struct {
	itsClient *ITSClient
}

func New(itsEndpoint, itsAPIKey string) *Handler {
	return &Handler{
		itsClient: NewITSClient(itsEndpoint, itsAPIKey),
	}
}

func (h *Handler) Name() string { return "ais140" }

// Detect identifies AIS140 packets by their '$' prefix (similar to NMEA).
func (h *Handler) Detect(header []byte) bool {
	return len(header) > 0 && header[0] == '$'
}

// Authenticate performs AIS140 device registration handshake.
// The first packet contains the vehicle registration number and device ID.
func (h *Handler) Authenticate(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("ais140 login read: %w", err)
	}

	// AIS140 login: $HLP,<VRN>,<IMEI>,<FW_VER>*<CHECKSUM>
	line = strings.TrimSpace(line)
	if !isValidChecksum(line) {
		return "", fmt.Errorf("ais140 login checksum invalid: %q", line)
	}

	fields := parseAIS140(line)
	if len(fields) < 3 || fields[0] != "HLP" {
		return "", fmt.Errorf("ais140 login unexpected packet: %q", line)
	}

	deviceID := fields[2] // IMEI
	// VRN (vehicle registration number) stored in fields[1] — available for enrichment

	// Send ACK
	ack := buildPacket("HBT", []string{"1"})
	if _, err := conn.Write([]byte(ack + "\r\n")); err != nil {
		return "", fmt.Errorf("ais140 send ack: %w", err)
	}

	return deviceID, nil
}

// ParsePacket reads one AIS140 packet from the connection.
func (h *Handler) ParsePacket(conn net.Conn) ([]protocol.ParsedRecord, []byte, error) {
	conn.SetDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, nil, fmt.Errorf("ais140 read: %w", err)
	}
	raw := []byte(line)
	line = strings.TrimSpace(line)

	if !isValidChecksum(line) {
		return nil, raw, fmt.Errorf("ais140 checksum invalid: %q", line)
	}

	fields := parseAIS140(line)
	if len(fields) == 0 {
		return nil, raw, fmt.Errorf("ais140 empty packet")
	}

	msgType := fields[0]
	switch msgType {
	case "NRM", "TRP": // Normal tracking, Trip data
		rec, err := parsePosition(fields)
		if err != nil {
			return nil, raw, err
		}
		return []protocol.ParsedRecord{rec}, raw, nil

	case "SOS", "EMR": // Emergency / SOS
		rec, err := parsePosition(fields)
		if err != nil {
			return nil, raw, err
		}
		rec.IOData[236] = 1 // SOS event IO ID
		// Forward emergency alert to ITS portal (non-blocking)
		go h.itsClient.ForwardEmergency(line) //nolint:errcheck
		return []protocol.ParsedRecord{rec}, raw, nil

	case "HBT": // Heartbeat
		// no position data, return empty
		return nil, raw, nil
	}

	return nil, raw, fmt.Errorf("ais140 unknown message type: %q", msgType)
}

func (h *Handler) BuildACK(_ []byte) []byte { return nil }

// ── Parsers ───────────────────────────────────────────────────────────────────

// parsePosition parses an AIS140 NRM (normal) packet.
// Field order: <type>,<date>,<time>,<lat>,<N/S>,<lng>,<E/W>,<speed>,<heading>,<status>,...
func parsePosition(fields []string) (protocol.ParsedRecord, error) {
	rec := protocol.ParsedRecord{IOData: make(map[int]int64)}
	if len(fields) < 9 {
		return rec, fmt.Errorf("ais140 NRM too few fields: %d", len(fields))
	}

	// Date: DDMMYYYY
	dateStr := fields[1]
	// Time: HHMMSS
	timeStr := fields[2]
	if len(dateStr) >= 8 && len(timeStr) >= 6 {
		day, _ := strconv.Atoi(dateStr[0:2])
		month, _ := strconv.Atoi(dateStr[2:4])
		year, _ := strconv.Atoi(dateStr[4:8])
		hour, _ := strconv.Atoi(timeStr[0:2])
		min, _ := strconv.Atoi(timeStr[2:4])
		sec, _ := strconv.Atoi(timeStr[4:6])
		ts := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
		rec.Timestamp = ts.UnixMilli()
	} else {
		rec.Timestamp = time.Now().UnixMilli()
	}

	// Latitude
	lat, _ := strconv.ParseFloat(fields[3], 64)
	if fields[4] == "S" {
		lat = -lat
	}
	rec.Lat = lat

	// Longitude
	lng, _ := strconv.ParseFloat(fields[5], 64)
	if fields[6] == "W" {
		lng = -lng
	}
	rec.Lng = lng

	// Speed (km/h)
	speed, _ := strconv.ParseFloat(fields[7], 64)
	rec.Speed = uint16(speed)

	// Heading
	heading, _ := strconv.ParseFloat(fields[8], 64)
	rec.Heading = uint16(heading)

	rec.Valid = true
	rec.RawCodec = 0
	return rec, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// parseAIS140 splits a line like "$NRM,field1,field2*XX" into fields (without '$' and checksum).
func parseAIS140(line string) []string {
	line = strings.TrimPrefix(line, "$")
	if idx := strings.LastIndex(line, "*"); idx != -1 {
		line = line[:idx]
	}
	return strings.Split(line, ",")
}

// isValidChecksum verifies the XOR checksum appended as *XX.
func isValidChecksum(line string) bool {
	line = strings.TrimPrefix(line, "$")
	starIdx := strings.LastIndex(line, "*")
	if starIdx < 0 || starIdx+3 > len(line) {
		return false
	}
	data := line[:starIdx]
	checksumStr := line[starIdx+1 : starIdx+3]

	var xor byte
	for _, c := range data {
		xor ^= byte(c)
	}
	expected := fmt.Sprintf("%02X", xor)
	return strings.EqualFold(checksumStr, expected)
}

func buildPacket(msgType string, fields []string) string {
	body := msgType + "," + strings.Join(fields, ",")
	var xor byte
	for _, c := range body {
		xor ^= byte(c)
	}
	return fmt.Sprintf("$%s*%02X", body, xor)
}
