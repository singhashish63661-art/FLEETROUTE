// Package main is the entry point for the WebSocket service.
// It provides real-time position and alert streams via WebSocket,
// backed by Redis Pub/Sub for multi-node fan-out.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	pkgauth "gpsgo/pkg/auth"
	pkgdb "gpsgo/pkg/db"
	"gpsgo/websocket-service/internal/hub"
	wshandler "gpsgo/websocket-service/internal/handler"
	"gpsgo/websocket-service/internal/redis_consumer"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb, err := pkgdb.NewRedis(ctx, pkgdb.RedisConfig{
		Addr:     envStr("REDIS_ADDR", "localhost:6379"),
		Password: envStr("REDIS_PASSWORD", ""),
	})
	if err != nil {
		logger.Fatal("Redis connect", zap.Error(err))
	}
	defer rdb.Close()

	// ── Auth ──────────────────────────────────────────────────────────────────
	authMgr, err := pkgauth.NewManager(
		envStr("JWT_PRIVATE_KEY_PATH", "secrets/jwt_private.pem"),
		envStr("JWT_PUBLIC_KEY_PATH", "secrets/jwt_public.pem"),
		15*time.Minute,
		7*24*time.Hour,
	)
	if err != nil {
		logger.Fatal("auth manager", zap.Error(err))
	}

	// ── Hub ───────────────────────────────────────────────────────────────────
	h := hub.New(logger)
	go h.Run(ctx)

	// ── Redis Pub/Sub → Hub ───────────────────────────────────────────────────
	consumer := redis_consumer.New(rdb, h, logger)
	go consumer.Subscribe(ctx)

	// ── HTTP / WebSocket routes ───────────────────────────────────────────────
	mux := http.NewServeMux()

	liveHandler := wshandler.NewLiveHandler(h, authMgr, logger)
	alertHandler := wshandler.NewAlertHandler(h, authMgr, logger)

	mux.HandleFunc("/ws/v1/live", liveHandler.ServeWS)
	mux.HandleFunc("/ws/v1/alerts", alertHandler.ServeWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	port := envStr("WS_PORT", "8081")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second, // long for WS connections
	}

	logger.Info("websocket service starting", zap.String("port", port))
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("WebSocket server", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down WebSocket service...")
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx) //nolint:errcheck
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
