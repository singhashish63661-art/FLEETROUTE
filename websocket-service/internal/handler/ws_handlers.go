package handler

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	pkgauth "gpsgo/pkg/auth"
	"gpsgo/websocket-service/internal/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: validate origin in production
	},
}

// LiveHandler serves the /ws/v1/live endpoint.
type LiveHandler struct {
	hub     *hub.Hub
	authMgr *pkgauth.Manager
	logger  *zap.Logger
}

func NewLiveHandler(h *hub.Hub, authMgr *pkgauth.Manager, logger *zap.Logger) *LiveHandler {
	return &LiveHandler{hub: h, authMgr: authMgr, logger: logger}
}

// ServeWS handles WebSocket upgrade for live position stream.
// Query params:
//   ?token=<jwt>              — authentication
//   ?devices=dev1,dev2,...    — optional device filter (empty = all tenant devices)
func (h *LiveHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Validate JWT from query param (headers aren't easy to set in WS)
	token := r.URL.Query().Get("token")
	if token == "" {
		// Also accept Authorization header
		auth := r.Header.Get("Authorization")
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.authMgr.Validate(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Device filter
	var deviceFilter []string
	if d := r.URL.Query().Get("devices"); d != "" {
		deviceFilter = strings.Split(d, ",")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("ws upgrade", zap.Error(err))
		return
	}

	client := h.hub.Register(conn, claims.TenantID, deviceFilter)

	h.logger.Info("ws client connected",
		zap.String("tenant", claims.TenantID),
		zap.String("user", claims.UserID),
		zap.Int("device_filter", len(deviceFilter)),
	)

	go h.hub.WritePump(client)
	h.hub.ReadPump(r.Context(), client)
}

// AlertHandler serves the /ws/v1/alerts endpoint.
type AlertHandler struct {
	hub     *hub.Hub
	authMgr *pkgauth.Manager
	logger  *zap.Logger
}

func NewAlertHandler(h *hub.Hub, authMgr *pkgauth.Manager, logger *zap.Logger) *AlertHandler {
	return &AlertHandler{hub: h, authMgr: authMgr, logger: logger}
}

func (h *AlertHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := h.authMgr.Validate(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := h.hub.Register(conn, claims.TenantID+"_alerts", nil)
	go h.hub.WritePump(client)
	h.hub.ReadPump(r.Context(), client)
}
