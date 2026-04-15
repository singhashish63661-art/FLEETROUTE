// Command injectsample publishes one synthetic ParsedRecord to JetStream (gpsgo.avl.raw)
// so stream-processor can enrich it, update Redis live state, and fan out over WebSocket.
//
// Usage (from repo root, with NATS reachable — e.g. docker compose):
//
//	cd stream-processor && NATS_URL=nats://127.0.0.1:4222 go run ./cmd/injectsample -device <DEVICE_UUID>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	natsclient "gpsgo/pkg/nats"
	"gpsgo/pkg/protocol"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222"
	}

	deviceID := flag.String("device", "", "Device UUID (from GET /api/v1/devices)")
	lat := flag.Float64("lat", 37.7749, "latitude (WGS84)")
	lng := flag.Float64("lng", -122.4194, "longitude (WGS84)")
	speed := flag.Uint("speed", 45, "speed km/h")
	flag.Parse()

	if *deviceID == "" {
		fmt.Fprintln(os.Stderr, "injectsample: -device <uuid> is required")
		fmt.Fprintln(os.Stderr, "example: NATS_URL=nats://127.0.0.1:4222 go run ./cmd/injectsample -device <uuid>")
		os.Exit(1)
	}

	ts := time.Now().UnixMilli()
	rec := protocol.ParsedRecord{
		DeviceID:   *deviceID,
		Timestamp:  ts,
		Lat:        *lat,
		Lng:        *lng,
		Altitude:   10,
		Speed:      uint16(*speed),
		Heading:    90,
		Satellites: 8,
		Valid:      true,
		Priority:   0,
		IOData:     map[int]int64{239: 1, 240: 1}, // ignition + movement (Teltonika-style)
		RawCodec:   8,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	nc, err := natsclient.New(natsURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nats:", err)
		os.Exit(1)
	}
	defer nc.Close()

	msgID := fmt.Sprintf("%s-%d", rec.DeviceID, rec.Timestamp)
	if err := nc.Publish(context.Background(), natsclient.SubjectRawAVL, msgID, data); err != nil {
		fmt.Fprintln(os.Stderr, "publish:", err)
		os.Exit(1)
	}

	fmt.Printf("Published one raw AVL to %s (device=%s lat=%.6f lng=%.6f)\n",
		natsclient.SubjectRawAVL, rec.DeviceID, rec.Lat, rec.Lng)
}
