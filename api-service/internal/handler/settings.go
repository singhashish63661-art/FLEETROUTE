package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	pkgauth "gpsgo/pkg/auth"
)

type SettingsHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewSettingsHandler(pool *pgxpool.Pool, logger *zap.Logger) *SettingsHandler {
	return &SettingsHandler{pool: pool, logger: logger}
}

func (h *SettingsHandler) Overview(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	userID := pkgauth.UserID(c)

	var (
		tenantIDOut  string
		tenantName   string
		tenantSlug   string
		tenantPlan   string
		settingsJSON string
		maxDevices   int
		tenantActive bool
	)
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT id::text, name, slug, plan, COALESCE(settings, '{}'::jsonb)::text, max_devices, is_active
		 FROM tenants
		 WHERE id=$1 AND deleted_at IS NULL`,
		tenantID,
	).Scan(&tenantIDOut, &tenantName, &tenantSlug, &tenantPlan, &settingsJSON, &maxDevices, &tenantActive)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load tenant")
		return
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		settings = map[string]any{}
	}

	var (
		userIDOut   string
		userEmail   string
		userName    string
		userRole    string
		userPhone   *string
		userActive  bool
		userCreated any
	)
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT id::text, email, name, role, phone, is_active, created_at
		 FROM users
		 WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		userID, tenantID,
	).Scan(&userIDOut, &userEmail, &userName, &userRole, &userPhone, &userActive, &userCreated)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load user")
		return
	}

	respondOK(c, gin.H{
		"tenant": gin.H{
			"id":          tenantIDOut,
			"name":        tenantName,
			"slug":        tenantSlug,
			"plan":        tenantPlan,
			"settings":    settings,
			"max_devices": maxDevices,
			"is_active":   tenantActive,
		},
		"user": gin.H{
			"id":         userIDOut,
			"email":      userEmail,
			"name":       userName,
			"role":       userRole,
			"phone":      userPhone,
			"is_active":  userActive,
			"created_at": userCreated,
		},
	})
}

func (h *SettingsHandler) Users(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id::text, email, name, role, phone, is_active, created_at
		 FROM users
		 WHERE tenant_id=$1 AND deleted_at IS NULL
		 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load users")
		return
	}
	defer rows.Close()

	var users []map[string]any
	for rows.Next() {
		var (
			id      string
			email   string
			name    string
			role    string
			phone   *string
			active  bool
			created any
		)
		if err := rows.Scan(&id, &email, &name, &role, &phone, &active, &created); err != nil {
			continue
		}
		users = append(users, map[string]any{
			"id":         id,
			"email":      email,
			"name":       name,
			"role":       role,
			"phone":      phone,
			"is_active":  active,
			"created_at": created,
		})
	}

	respondOK(c, users)
}

func (h *SettingsHandler) UpdatePreferences(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	var body struct {
		SpeedLimitKmh  int `json:"speed_limit_kmh"`
		IdleTimeoutMin int `json:"idle_timeout_mins"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if body.SpeedLimitKmh <= 0 || body.IdleTimeoutMin <= 0 {
		respondError(c, http.StatusBadRequest, "speed_limit_kmh and idle_timeout_mins must be > 0")
		return
	}

	var settings any
	err := h.pool.QueryRow(context.Background(),
		`UPDATE tenants
		 SET settings = COALESCE(settings, '{}'::jsonb) ||
		   jsonb_build_object('speed_limit_kmh', $2, 'idle_timeout_mins', $3),
		     updated_at = now()
		 WHERE id=$1 AND deleted_at IS NULL
		 RETURNING settings`,
		tenantID, body.SpeedLimitKmh, body.IdleTimeoutMin,
	).Scan(&settings)
	if err != nil {
		h.logger.Error("update tenant preferences", zap.Error(err))
		respondError(c, http.StatusInternalServerError, "failed to update preferences")
		return
	}

	respondOK(c, gin.H{"settings": settings})
}
