package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	pkgauth "gpsgo/pkg/auth"
)

// VehicleHandler handles vehicle CRUD and command dispatch.
type VehicleHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewVehicleHandler(pool *pgxpool.Pool, logger *zap.Logger) *VehicleHandler {
	return &VehicleHandler{pool: pool, logger: logger}
}

func (h *VehicleHandler) List(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	rows, _ := h.pool.Query(c.Request.Context(),
		`SELECT id, tenant_id, registration, make, model, year, device_id, created_at
		 FROM vehicles WHERE tenant_id=$1 AND deleted_at IS NULL ORDER BY registration`,
		tenantID,
	)
	defer rows.Close()
	var vs []map[string]any
	for rows.Next() {
		vals, _ := rows.Values()
		fields := []string{"id", "tenant_id", "registration", "make", "model", "year", "device_id", "created_at"}
		m := make(map[string]any)
		for i, f := range fields {
			if i < len(vals) {
				m[f] = vals[i]
			}
		}
		vs = append(vs, m)
	}
	respondOK(c, vs)
}

func (h *VehicleHandler) Get(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	id := c.Param("id")
	rows, _ := h.pool.Query(c.Request.Context(),
		`SELECT id, tenant_id, registration, make, model, year, device_id, created_at
		 FROM vehicles WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		id, tenantID,
	)
	defer rows.Close()
	if !rows.Next() {
		respondError(c, http.StatusNotFound, "vehicle not found")
		return
	}
	vals, _ := rows.Values()
	respondOK(c, vals)
}

func (h *VehicleHandler) Create(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	var body struct {
		Registration string  `json:"registration" binding:"required"`
		Make         string  `json:"make"`
		Model        string  `json:"model"`
		Year         int     `json:"year"`
		DeviceID     *string `json:"device_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var id string
	h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO vehicles (tenant_id, registration, make, model, year, device_id)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		tenantID, body.Registration, body.Make, body.Model, body.Year, body.DeviceID,
	).Scan(&id)
	respondCreated(c, gin.H{"id": id})
}

func (h *VehicleHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNoContent, nil)
}
func (h *VehicleHandler) Delete(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	h.pool.Exec(c.Request.Context(), //nolint:errcheck
		`UPDATE vehicles SET deleted_at=now() WHERE id=$1 AND tenant_id=$2`,
		c.Param("id"), tenantID,
	)
	c.JSON(http.StatusNoContent, nil)
}

// SendCommand godoc
// @Summary      Send a command to a vehicle (immobilize, unlock, etc.)
// @Tags         vehicles
// @Security     BearerAuth
// @Router       /vehicles/{id}/command [post]
func (h *VehicleHandler) SendCommand(c *gin.Context) {
	var body struct {
		Command string            `json:"command" binding:"required"`
		Params  map[string]string `json:"params"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	// TODO: look up device for vehicle, dispatch command via connection registry
	// Commands: "immobilize", "unlock", "request_config", "reboot"
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"status": "queued", "command": body.Command}})
}

// ── GeofenceHandler ───────────────────────────────────────────────────────────

type GeofenceHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewGeofenceHandler(pool *pgxpool.Pool, logger *zap.Logger) *GeofenceHandler {
	return &GeofenceHandler{pool: pool, logger: logger}
}
func (h *GeofenceHandler) List(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	rows, _ := h.pool.Query(c.Request.Context(),
		`SELECT id, name, shape_type, ST_AsGeoJSON(geometry)::text, created_at
		 FROM geofences WHERE tenant_id=$1 AND deleted_at IS NULL`,
		tenantID,
	)
	defer rows.Close()
	var gs []map[string]any
	for rows.Next() {
		vals, _ := rows.Values()
		m := map[string]any{"id": vals[0], "name": vals[1], "shape_type": vals[2],
			"geometry": vals[3], "created_at": vals[4]}
		gs = append(gs, m)
	}
	respondOK(c, gs)
}
func (h *GeofenceHandler) Create(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	var body struct {
		Name      string `json:"name" binding:"required"`
		ShapeType string `json:"shape_type" binding:"required"` // circle, polygon, corridor
		GeoJSON   string `json:"geojson" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var id string
	h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO geofences (tenant_id, name, shape_type, geometry)
		 VALUES ($1, $2, $3, ST_GeomFromGeoJSON($4)) RETURNING id`,
		tenantID, body.Name, body.ShapeType, body.GeoJSON,
	).Scan(&id)
	respondCreated(c, gin.H{"id": id})
}
func (h *GeofenceHandler) Get(c *gin.Context)    { respondOK(c, gin.H{}) }
func (h *GeofenceHandler) Update(c *gin.Context)  { c.JSON(http.StatusNoContent, nil) }
func (h *GeofenceHandler) Delete(c *gin.Context)  { c.JSON(http.StatusNoContent, nil) }
func (h *GeofenceHandler) Events(c *gin.Context)  { respondOK(c, []any{}) }

// ── AlertHandler ──────────────────────────────────────────────────────────────

type AlertHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewAlertHandler(pool *pgxpool.Pool, logger *zap.Logger) *AlertHandler {
	return &AlertHandler{pool: pool, logger: logger}
}
func (h *AlertHandler) List(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	rows, _ := h.pool.Query(c.Request.Context(),
		`SELECT id, device_id, rule_id, alert_type, severity, message, triggered_at, acknowledged_at
		 FROM alerts WHERE tenant_id=$1 ORDER BY triggered_at DESC LIMIT 100`,
		tenantID,
	)
	defer rows.Close()
	var alerts []map[string]any
	for rows.Next() {
		vals, _ := rows.Values()
		fields := []string{"id", "device_id", "rule_id", "alert_type", "severity", "message", "triggered_at", "acknowledged_at"}
		m := make(map[string]any)
		for i, f := range fields {
			if i < len(vals) {
				m[f] = vals[i]
			}
		}
		alerts = append(alerts, m)
	}
	respondOK(c, alerts)
}
func (h *AlertHandler) Acknowledge(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	alertID := c.Param("id")
	userID := pkgauth.UserID(c)
	h.pool.Exec(c.Request.Context(), //nolint:errcheck
		`UPDATE alerts SET acknowledged_at=now(), acknowledged_by=$3
		 WHERE id=$1 AND tenant_id=$2`,
		alertID, tenantID, userID,
	)
	c.JSON(http.StatusNoContent, nil)
}

// ── RuleHandler ───────────────────────────────────────────────────────────────

type RuleHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewRuleHandler(pool *pgxpool.Pool, logger *zap.Logger) *RuleHandler {
	return &RuleHandler{pool: pool, logger: logger}
}
func (h *RuleHandler) List(c *gin.Context)      { respondOK(c, []any{}) }
func (h *RuleHandler) Create(c *gin.Context)    { respondCreated(c, gin.H{"id": "new"}) }
func (h *RuleHandler) Get(c *gin.Context)       { respondOK(c, gin.H{}) }
func (h *RuleHandler) Update(c *gin.Context)    { c.JSON(http.StatusNoContent, nil) }
func (h *RuleHandler) Delete(c *gin.Context)    { c.JSON(http.StatusNoContent, nil) }
func (h *RuleHandler) Templates(c *gin.Context) {
	respondOK(c, []gin.H{
		{"id": "overspeed", "name": "Overspeed Alert", "description": "Alert when speed exceeds threshold"},
		{"id": "geofence_entry", "name": "Geofence Entry", "description": "Alert on geofence entry"},
		{"id": "geofence_exit", "name": "Geofence Exit", "description": "Alert on geofence exit"},
		{"id": "idle_detection", "name": "Excessive Idling", "description": "Alert when ignition ON + speed=0 for N minutes"},
		{"id": "fuel_theft", "name": "Fuel Theft Detection", "description": "Alert on rapid fuel level drop"},
		{"id": "harsh_driving", "name": "Harsh Driving", "description": "Alert on harsh acceleration/braking"},
		{"id": "device_tamper", "name": "Device Tamper", "description": "Alert on power cut or tamper detection"},
	})
}

// ── ReportHandler ─────────────────────────────────────────────────────────────

type ReportHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewReportHandler(pool *pgxpool.Pool, logger *zap.Logger) *ReportHandler {
	return &ReportHandler{pool: pool, logger: logger}
}
func (h *ReportHandler) Trips(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	rows, _ := h.pool.Query(c.Request.Context(),
		`SELECT id, device_id, started_at, ended_at, distance_m, duration_s, max_speed 
		 FROM trips WHERE tenant_id=$1 ORDER BY started_at DESC LIMIT 50`,
		tenantID,
	)
	defer rows.Close()
	var trips []map[string]any
	for rows.Next() {
		vals, _ := rows.Values()
		trips = append(trips, map[string]any{
			"id": vals[0], "device_id": vals[1], "started_at": vals[2],
			"ended_at": vals[3], "distance_m": vals[4], "duration_s": vals[5],
			"max_speed": vals[6],
		})
	}
	respondOK(c, trips)
}

func (h *ReportHandler) Fuel(c *gin.Context) { 
	respondOK(c, []map[string]any{
		{"date": "2023-10-01", "liters": 45.2, "cost": 65.0},
		{"date": "2023-10-02", "liters": 12.5, "cost": 18.0},
	})
}

func (h *ReportHandler) DriverBehavior(c *gin.Context) {
	respondOK(c, []map[string]any{
		{"driver": "John Doe", "score": 92, "events": 3},
		{"driver": "Jane Smith", "score": 85, "events": 8},
	})
}

func (h *ReportHandler) GeofenceViolations(c *gin.Context) { 
	respondOK(c, []map[string]any{})
}

// ── DriverHandler ─────────────────────────────────────────────────────────────

type DriverHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewDriverHandler(pool *pgxpool.Pool, logger *zap.Logger) *DriverHandler {
	return &DriverHandler{pool: pool, logger: logger}
}
func (h *DriverHandler) List(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	rows, _ := h.pool.Query(c.Request.Context(),
		`SELECT id, name, license_number, rfid_uid, phone, created_at
		 FROM drivers WHERE tenant_id=$1 AND deleted_at IS NULL ORDER BY name`,
		tenantID,
	)
	defer rows.Close()
	var ds []map[string]any
	for rows.Next() {
		vals, _ := rows.Values()
		fields := []string{"id", "name", "license_number", "rfid_uid", "phone", "created_at"}
		m := make(map[string]any)
		for i, f := range fields {
			if i < len(vals) {
				m[f] = vals[i]
			}
		}
		ds = append(ds, m)
	}
	respondOK(c, ds)
}
func (h *DriverHandler) Create(c *gin.Context) { respondCreated(c, gin.H{"id": "new"}) }
func (h *DriverHandler) Get(c *gin.Context)    { respondOK(c, gin.H{}) }
func (h *DriverHandler) Score(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	driverID := c.Param("id")

	row := h.pool.QueryRow(c.Request.Context(), `
		SELECT 
			COALESCE(COUNT(id), 0),
			COALESCE(SUM(duration_s), 0),
			COALESCE(SUM(harsh_accel), 0),
			COALESCE(SUM(harsh_brake), 0),
			COALESCE(SUM(overspeed_count), 0)
		FROM trips 
		WHERE tenant_id=$1 AND driver_id=$2
	`, tenantID, driverID)

	var trips, duration, hAccel, hBrake, overspeed int
	if err := row.Scan(&trips, &duration, &hAccel, &hBrake, &overspeed); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to compute score")
		return
	}

	score := 100 - (hAccel*2 + hBrake*2 + overspeed*1)
	if score < 0 { score = 0 }

	respondOK(c, gin.H{
		"score": score,
		"period": "all_time",
		"trips": trips,
		"duration_s": duration,
		"overspeed": overspeed,
		"harsh_accel": hAccel,
		"harsh_brake": hBrake,
	})
}
