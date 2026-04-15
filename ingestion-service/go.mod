module gpsgo/ingestion-service

go 1.22

require (
	gpsgo/pkg v0.0.0
	gpsgo/protocols v0.0.0
	github.com/nats-io/nats.go v1.35.0
	go.uber.org/zap v1.27.0
)

replace (
	gpsgo/pkg => ../pkg
	gpsgo/protocols => ../protocols
)
