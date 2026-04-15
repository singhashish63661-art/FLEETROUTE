// Package main is the entry point for the REST API service.
//
// @title           GPS Fleet Management API
// @version         1.0
// @description     Multi-tenant GPS fleet management platform REST API
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"gpsgo/api-service/internal/router"
	pkgauth "gpsgo/pkg/auth"
	pkgdb "gpsgo/pkg/db"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Database ──────────────────────────────────────────────────────────────
	timescaleDSN := envStr("TIMESCALE_DSN", "")
	pool, err := pkgdb.NewPool(ctx, timescaleDSN)
	if err != nil {
		logger.Fatal("TimescaleDB connect", zap.Error(err))
	}
	defer pool.Close()

	rdb, err := pkgdb.NewRedis(ctx, pkgdb.RedisConfig{
		Addr:     envStr("REDIS_ADDR", "localhost:6379"),
		Password: envStr("REDIS_PASSWORD", ""),
	})
	if err != nil {
		logger.Fatal("Redis connect", zap.Error(err))
	}
	defer rdb.Close()

	// ── Auth Manager ──────────────────────────────────────────────────────────
	authMgr, err := pkgauth.NewManager(
		envStr("JWT_PRIVATE_KEY_PATH", "secrets/jwt_private.pem"),
		envStr("JWT_PUBLIC_KEY_PATH", "secrets/jwt_public.pem"),
		15*time.Minute,
		7*24*time.Hour,
	)
	if err != nil {
		logger.Fatal("JWT manager init", zap.Error(err))
	}

	// ── HTTP Server ───────────────────────────────────────────────────────────
	r := router.New(pool, rdb, authMgr, logger, timescaleDSN)

	port := envStr("API_PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("API service starting", zap.String("port", port))
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down API service...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx) //nolint:errcheck
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
