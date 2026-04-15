// Package tk103 implements the TK103 GPS tracker protocol handler.
// Supports both ASCII (text) and binary variants.
//
// ASCII format example:
//   ##,imei:123456789012345,A;
//   (123456789012345,normal,020614195856.000,A,2234.8197,N,11354.4926,E,0.00,0.00,;
package tk103

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"gpsgo/pkg/protocol"
)

// Handler implements protocol.Handler for TK103 devices.
type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Name() string { return "tk103" }

// Detect identifies TK103 by its ASCII login prefix "##,imei:".
func (h *Handler) Detect(header []byte) bool {
	return strings.HasPrefix(string(header), "##")
}

// Authenticate handles the TK103 login message: ##,imei:IMEI,A;
func (h *Handler) Authenticate(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString(';')
	if err != nil {
		return "", fmt.Errorf("tk103 login read: %w", err)
	}

	// Expected: ##,imei:123456789012345,A;
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "##,imei:") {
		return "", fmt.Errorf("tk103 invalid login: %q", line)
	}
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return "", fmt.Errorf("tk103 login malformed: %q", line)
	}
	imei := strings.TrimPrefix(parts[1], "imei:")

	// Send ACK: LOAD
	if _, err := conn.Write([]byte("LOAD")); err != nil {
		return "", fmt.Errorf("tk103 send LOAD: %w", err)
	}

	return imei, nil
}

// ParsePacket reads one TK103 position report and returns parsed records.
func (h *Handler) ParsePacket(conn net.Conn) ([]protocol.ParsedRecord, []byte, error) {
	conn.SetDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
	defer conn.SetDeadline(time.Time{})                //nolint:errcheck

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString(';')
	if err != nil && err != io.EOF {
		return nil, nil, fmt.Errorf("tk103 read: %w", err)
	}
	raw := []byte(line)
	line = strings.TrimSpace(line)

	rec, err := parseASCIILine(line)
	if err != nil {
		return nil, raw, fmt.Errorf("tk103 parse: %w", err)
	}

	return []protocol.ParsedRecord{rec}, raw, nil
}

func (h *Handler) BuildACK(_ []byte) []byte { return nil }

// parseASCIILine parses a TK103 ASCII position report.
// Format: (IMEI,type,DDMMYYHHMMSS.sss,A/V,lat,N/S,lng,E/W,speed,heading,;
func parseASCIILine(line string) (protocol.ParsedRecord, error) {
	rec := protocol.ParsedRecord{IOData: make(map[int]int64)}

	// Strip surrounding parens if present
	line = strings.Trim(line, "();")
	parts := strings.Split(line, ",")
	if len(parts) < 9 {
		return rec, fmt.Errorf("tk103: too few fields: %d", len(parts))
	}

	// fields: 0=IMEI, 1=type, 2=datetime, 3=valid, 4=lat, 5=N/S, 6=lng, 7=E/W, 8=speed, 9=heading
	dateStr := parts[2]
	var ts time.Time
	if len(dateStr) >= 12 {
		day, _ := strconv.Atoi(dateStr[0:2])
		month, _ := strconv.Atoi(dateStr[2:4])
		year := 2000 + mustAtoi(dateStr[4:6])
		hour, _ := strconv.Atoi(dateStr[6:8])
		min, _ := strconv.Atoi(dateStr[8:10])
		sec, _ := strconv.Atoi(dateStr[10:12])
		ts = time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
	} else {
		ts = time.Now().UTC()
	}
	rec.Timestamp = ts.UnixMilli()

	valid := strings.TrimSpace(parts[3]) == "A"
	rec.Valid = valid

	lat, _ := parseNMEACoord(parts[4])
	if strings.TrimSpace(parts[5]) == "S" {
		lat = -lat
	}
	rec.Lat = lat

	lng, _ := parseNMEACoord(parts[6])
	if strings.TrimSpace(parts[7]) == "W" {
		lng = -lng
	}
	rec.Lng = lng

	speed, _ := strconv.ParseFloat(strings.TrimSpace(parts[8]), 64)
	rec.Speed = uint16(speed * 1.852) // knots → km/h

	if len(parts) > 9 {
		heading, _ := strconv.ParseFloat(strings.TrimSpace(parts[9]), 64)
		rec.Heading = uint16(heading)
	}

	rec.RawCodec = 0
	return rec, nil
}

// parseNMEACoord converts NMEA lat/lng string (DDDMM.MMMM) to decimal degrees.
func parseNMEACoord(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	// Find decimal point
	dotIdx := strings.Index(s, ".")
	if dotIdx < 2 {
		return 0, fmt.Errorf("invalid NMEA coord: %q", s)
	}
	degStr := s[:dotIdx-2]
	minStr := s[dotIdx-2:]
	deg, err := strconv.ParseFloat(degStr, 64)
	if err != nil {
		return 0, err
	}
	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return 0, err
	}
	return math.Round((deg+min/60.0)*1e7) / 1e7, nil
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
