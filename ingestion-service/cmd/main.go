// Package main is the entry point for the ingestion service.
// It starts TCP listeners on all configured protocol ports and publishes
// parsed AVL records to NATS JetStream.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"gpsgo/ingestion-service/config"
	"gpsgo/ingestion-service/internal/server"
	natsclient "gpsgo/pkg/nats"
	"gpsgo/pkg/protocol"
	"gpsgo/protocols/ais140"
	"gpsgo/protocols/gt06"
	"gpsgo/protocols/jt808"
	"gpsgo/protocols/teltonika"
	"gpsgo/protocols/tk103"
)

func main() {
	cfg := config.Load()

	// ── Logger ────────────────────────────────────────────────────────────────
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	logger.Info("ingestion-service starting",
		zap.Int("port_gt06", cfg.PortGT06),
		zap.Int("port_tk103", cfg.PortTK103),
		zap.Int("port_ais140", cfg.PortAIS140),
		zap.Int("port_teltonika", cfg.PortTeltonikaTC),
	)

	// ── NATS JetStream ────────────────────────────────────────────────────────
	nc, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		logger.Fatal("NATS connect failed", zap.Error(err))
	}
	defer nc.Close()

	// ── Connection Registry ───────────────────────────────────────────────────
	registry := server.NewConnRegistry()

	// ── Record Publisher ──────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onRecord := func(deviceID string, records []protocol.ParsedRecord) {
		for _, rec := range records {
			data, err := json.Marshal(rec)
			if err != nil {
				logger.Error("marshal record", zap.String("device_id", deviceID), zap.Error(err))
				continue
			}
			msgID := fmt.Sprintf("%s-%d", deviceID, rec.Timestamp)
			if err := nc.Publish(ctx, natsclient.SubjectRawAVL, msgID, data); err != nil {
				logger.Error("publish record", zap.String("device_id", deviceID), zap.Error(err))
			}
		}
		logger.Debug("published records",
			zap.String("device_id", deviceID),
			zap.Int("count", len(records)),
		)
	}

	// ── Protocol Handlers → TCP Servers ───────────────────────────────────────
	type portHandler struct {
		port    int
		handler protocol.Handler
	}

	handlers := []portHandler{
		{cfg.PortGT06, gt06.New()},
		{cfg.PortTK103, tk103.New()},
		{cfg.PortJT808, jt808.New()},
		{cfg.PortAIS140, ais140.New(cfg.AIS140ITSEndpoint, cfg.AIS140ITSAPIKey)},
		{cfg.PortTeltonikaTC, teltonika.New()},
	}

	var wg sync.WaitGroup
	for _, ph := range handlers {
		srv := server.NewTCPServer(ph.port, ph.handler, registry, onRecord, logger)
		wg.Add(1)
		go func(s *server.TCPServer, name string) {
			defer wg.Done()
			if err := s.ListenAndServe(ctx); err != nil {
				logger.Error("listener error", zap.String("protocol", name), zap.Error(err))
			}
		}(srv, ph.handler.Name())
	}

	// Status reporter
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Info("connection stats", zap.Int("active_connections", registry.Count()))
			}
		}
	}()

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutdown signal received, draining connections...")
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("all connections drained, exiting cleanly")
	case <-time.After(30 * time.Second):
		logger.Warn("drain timeout exceeded, forcing exit")
	}
}
