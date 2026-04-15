package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all ingestion service configuration.
type Config struct {
	// Port assignments per protocol
	PortGT06          int
	PortTK103         int
	PortJT808         int
	PortAIS140        int
	PortTeltonikaTC   int
	PortTeltonikaUDP  int

	// NATS
	NATSUrl string

	// Observability
	LogLevel          string
	OTELEndpoint      string

	// Timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// AIS140
	AIS140ITSEndpoint string
	AIS140ITSAPIKey   string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		PortGT06:         envInt("PORT_GT06", 5001),
		PortTK103:        envInt("PORT_TK103", 5002),
		PortJT808:        envInt("PORT_JT808", 5003),
		PortAIS140:       envInt("PORT_AIS140", 5004),
		PortTeltonikaTC:  envInt("PORT_TELTONIKA_TCP", 5027),
		PortTeltonikaUDP: envInt("PORT_TELTONIKA_UDP", 5028),

		NATSUrl:  envStr("NATS_URL", "nats://localhost:4222"),
		LogLevel: envStr("LOG_LEVEL", "info"),
		OTELEndpoint: envStr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),

		ReadTimeout:  60 * time.Second,
		WriteTimeout: 10 * time.Second,

		AIS140ITSEndpoint: envStr("AIS140_ITS_ENDPOINT", ""),
		AIS140ITSAPIKey:   envStr("AIS140_ITS_API_KEY", ""),
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
