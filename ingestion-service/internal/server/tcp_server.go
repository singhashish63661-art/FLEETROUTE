package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"

	"gpsgo/pkg/protocol"
)

// TCPServer listens on a single TCP port and serves connections using the given protocol handler.
// Each connection runs in its own goroutine with context-aware cancellation.
type TCPServer struct {
	port     int
	handler  protocol.Handler
	registry *ConnRegistry
	onRecord func(deviceID string, records []protocol.ParsedRecord)
	logger   *zap.Logger
}

// NewTCPServer creates a TCPServer for the given protocol+port+record callback.
func NewTCPServer(
	port int,
	handler protocol.Handler,
	registry *ConnRegistry,
	onRecord func(deviceID string, records []protocol.ParsedRecord),
	logger *zap.Logger,
) *TCPServer {
	return &TCPServer{
		port:     port,
		handler:  handler,
		registry: registry,
		onRecord: onRecord,
		logger:   logger,
	}
}

// ListenAndServe starts the TCP server. It blocks until ctx is cancelled.
func (s *TCPServer) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp listen %s: %w", addr, err)
	}
	s.logger.Info("TCP listener started",
		zap.String("protocol", s.handler.Name()),
		zap.String("addr", addr),
	)

	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		ln.Close() // unblock Accept
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// Context cancelled — wait for all connections to drain
				wg.Wait()
				return nil
			default:
				s.logger.Warn("accept error", zap.String("protocol", s.handler.Name()), zap.Error(err))
				continue
			}
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleConn(ctx, conn)
		}()
	}
}

func (s *TCPServer) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()

	s.logger.Info("new connection",
		zap.String("protocol", s.handler.Name()),
		zap.String("remote", remoteAddr),
	)

	// ── Peek first bytes for auto-detection (optional) ───────────────────────
	// (Already dispatched by port, so handler is pre-selected here.)

	// ── Authenticate ─────────────────────────────────────────────────────────
	deviceID, err := s.handler.Authenticate(conn)
	if err != nil {
		s.logger.Warn("authentication failed",
			zap.String("protocol", s.handler.Name()),
			zap.String("remote", remoteAddr),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("device authenticated",
		zap.String("protocol", s.handler.Name()),
		zap.String("device_id", deviceID),
		zap.String("remote", remoteAddr),
	)

	// Register connection for bidirectional commands
	s.registry.Register(deviceID, conn)
	defer s.registry.Unregister(deviceID)

	// ── Parse loop ────────────────────────────────────────────────────────────
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		records, raw, err := s.handler.ParsePacket(conn)
		if err != nil {
			if isClosedOrEOF(err) {
				s.logger.Info("connection closed",
					zap.String("device_id", deviceID),
					zap.String("protocol", s.handler.Name()),
				)
			} else {
				s.logger.Error("parse error",
					zap.String("device_id", deviceID),
					zap.String("protocol", s.handler.Name()),
					zap.Error(err),
					zap.Int("raw_bytes", len(raw)),
				)
				// TODO: publish raw to dead-letter queue
			}
			return
		}

		if len(records) > 0 {
			// Set device ID on all records
			for i := range records {
				records[i].DeviceID = deviceID
			}
			s.onRecord(deviceID, records)
		}
	}
}

func isClosedOrEOF(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return containsAny(s, "EOF", "closed", "reset by peer", "broken pipe")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
