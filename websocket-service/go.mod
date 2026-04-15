module gpsgo/websocket-service

go 1.22

require (
	gpsgo/pkg v0.0.0
	github.com/gin-gonic/gin v1.10.0
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.5.3
	go.uber.org/zap v1.27.0
)

replace gpsgo/pkg => ../pkg
