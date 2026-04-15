package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkgauth "gpsgo/pkg/auth"
	pkgdb "gpsgo/pkg/db"
)

// DeviceHandler handles device CRUD, live position, and history endpoints.
type DeviceHandler struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	logger *zap.Logger
}

func NewDeviceHandler(pool *pgxpool.Pool, rdb *redis.Client, logger *zap.Logger) *DeviceHandler {
	return &DeviceHandler{pool: pool, rdb: rdb, logger: logger}
}

// Device is the API representation of a GPS device.
type Device struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	IMEI       string    `json:"imei"`
	Name       string    `json:"name"`
	Protocol   string    `json:"protocol"`
	VehicleID  *string   `json:"vehicle_id"`
	Online     bool      `json:"online"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// List godoc
// @Summary      List devices for tenant
// @Tags         devices
// @Security     BearerAuth
// @Produce      json
// @Success      200 {array} Device
// @Router       /devices [get]
func (h *DeviceHandler) List(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, tenant_id, imei, name, protocol, vehicle_id, last_seen_at, created_at
		 FROM devices WHERE tenant_id=$1 AND deleted_at IS NULL
		 ORDER BY name LIMIT 1000`,
		tenantID,
	)
	if err != nil {
		h.logger.Error("devices list", zap.Error(err))
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.TenantID, &d.IMEI, &d.Name, &d.Protocol,
			&d.VehicleID, &d.LastSeenAt, &d.CreatedAt); err != nil {
			continue
		}
		// Check online status from Redis
		key := pkgdb.KeyDeviceLive(tenantID, d.ID)
		d.Online = h.rdb.Exists(c.Request.Context(), key).Val() > 0
		devices = append(devices, d)
	}
	respondOK(c, devices)
}

// Get godoc
// @Summary      Get device by ID
// @Tags         devices
// @Security     BearerAuth
// @Param        id path string true "Device ID"
// @Router       /devices/{id} [get]
func (h *DeviceHandler) Get(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")

	var d Device
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT id, tenant_id, imei, name, protocol, vehicle_id, last_seen_at, created_at
		 FROM devices WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		deviceID, tenantID,
	).Scan(&d.ID, &d.TenantID, &d.IMEI, &d.Name, &d.Protocol, &d.VehicleID, &d.LastSeenAt, &d.CreatedAt)
	if err != nil {
		respondError(c, http.StatusNotFound, "device not found")
		return
	}
	key := pkgdb.KeyDeviceLive(tenantID, d.ID)
	d.Online = h.rdb.Exists(c.Request.Context(), key).Val() > 0
	respondOK(c, d)
}

// Create godoc
// @Summary      Register a new device
// @Tags         devices
// @Security     BearerAuth
// @Router       /devices [post]
func (h *DeviceHandler) Create(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	var body struct {
		IMEI      string  `json:"imei" binding:"required"`
		Name      string  `json:"name" binding:"required"`
		Protocol  string  `json:"protocol" binding:"required"`
		VehicleID *string `json:"vehicle_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var id string
	err := h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO devices (tenant_id, imei, name, protocol, vehicle_id)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		tenantID, body.IMEI, body.Name, body.Protocol, body.VehicleID,
	).Scan(&id)
	if err != nil {
		h.logger.Error("device create", zap.Error(err))
		respondError(c, http.StatusInternalServerError, "failed to create device")
		return
	}
	respondCreated(c, gin.H{"id": id})
}

// Update godoc
// @Summary      Update device
// @Tags         devices
// @Security     BearerAuth
// @Router       /devices/{id} [put]
func (h *DeviceHandler) Update(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")
	var body struct {
		Name      string  `json:"name"`
		VehicleID *string `json:"vehicle_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	_, err := h.pool.Exec(c.Request.Context(),
		`UPDATE devices SET name=$3, vehicle_id=$4, updated_at=now()
		 WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		deviceID, tenantID, body.Name, body.VehicleID,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// Delete godoc
// @Summary      Soft-delete a device
// @Tags         devices
// @Security     BearerAuth
// @Router       /devices/{id} [delete]
func (h *DeviceHandler) Delete(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")
	h.pool.Exec(c.Request.Context(), //nolint:errcheck
		`UPDATE devices SET deleted_at=now() WHERE id=$1 AND tenant_id=$2`,
		deviceID, tenantID,
	)
	c.JSON(http.StatusNoContent, nil)
}

// Live godoc
// @Summary      Get live position from Redis
// @Tags         devices
// @Security     BearerAuth
// @Param        id path string true "Device ID"
// @Router       /devices/{id}/live [get]
func (h *DeviceHandler) Live(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")
	key := pkgdb.KeyDeviceLive(tenantID, deviceID)

	data, err := h.rdb.Get(c.Request.Context(), key).Bytes()
	if err != nil {
		respondError(c, http.StatusNotFound, "device offline or not found")
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// History godoc
// @Summary      Get historical AVL records
// @Tags         devices
// @Security     BearerAuth
// @Param        id   path  string true  "Device ID"
// @Param        from query string false "From time (RFC3339)"
// @Param        to   query string false "To time (RFC3339)"
// @Router       /devices/{id}/history [get]
func (h *DeviceHandler) History(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")

	from, to := parseTimeRange(c)

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT timestamp, lat, lng, altitude, speed, heading, satellites, valid,
		        ignition, movement, external_voltage, battery_level
		 FROM avl_records
		 WHERE device_id=$1 AND tenant_id=$2 AND timestamp BETWEEN $3 AND $4
		 ORDER BY timestamp ASC LIMIT 5000`,
		deviceID, tenantID, from, to,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type Point struct {
		Timestamp       time.Time `json:"ts"`
		Lat             float64   `json:"lat"`
		Lng             float64   `json:"lng"`
		Altitude        int       `json:"alt"`
		Speed           int       `json:"spd"`
		Heading         int       `json:"hdg"`
		Satellites      int       `json:"sat"`
		Valid           bool      `json:"valid"`
		Ignition        bool      `json:"ign"`
		Movement        bool      `json:"mov"`
		ExternalVoltage float64   `json:"ext_v"`
		BatteryLevel    int       `json:"bat_pct"`
	}

	var points []Point
	for rows.Next() {
		var p Point
		rows.Scan(&p.Timestamp, &p.Lat, &p.Lng, &p.Altitude, &p.Speed, &p.Heading, //nolint:errcheck
			&p.Satellites, &p.Valid, &p.Ignition, &p.Movement, &p.ExternalVoltage, &p.BatteryLevel)
		points = append(points, p)
	}
	respondOK(c, points)
}

// Trips godoc
// @Summary      Get trips for device
// @Tags         devices
// @Security     BearerAuth
// @Router       /devices/{id}/trips [get]
func (h *DeviceHandler) Trips(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")
	from, to := parseTimeRange(c)

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, started_at, ended_at, start_lat, start_lng, end_lat, end_lng,
		        distance_m, duration_s, max_speed, avg_speed, idle_time_s
		 FROM trips
		 WHERE device_id=$1 AND tenant_id=$2 AND started_at BETWEEN $3 AND $4
		 ORDER BY started_at DESC LIMIT 100`,
		deviceID, tenantID, from, to,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var trips []map[string]any
	for rows.Next() {
		// scan into map for flexibility
		vals, _ := rows.Values()
		fields := []string{"id", "started_at", "ended_at", "start_lat", "start_lng",
			"end_lat", "end_lng", "distance_m", "duration_s", "max_speed", "avg_speed", "idle_time_s"}
		m := make(map[string]any, len(fields))
		for i, f := range fields {
			if i < len(vals) {
				m[f] = vals[i]
			}
		}
		trips = append(trips, m)
	}
	respondOK(c, trips)
}

// Telemetry godoc
// @Summary      Get latest telemetry snapshot
// @Tags         devices
// @Security     BearerAuth
// @Router       /devices/{id}/telemetry [get]
func (h *DeviceHandler) Telemetry(c *gin.Context) {
	tenantID := pkgauth.TenantID(c)
	deviceID := c.Param("id")

	row := h.pool.QueryRow(c.Request.Context(),
		`SELECT timestamp, ignition, movement, external_voltage, battery_voltage, battery_level,
		        gnss_status, gsm_signal, engine_rpm, fuel_level, temperature_1, sos_event, io_data
		 FROM avl_records
		 WHERE device_id=$1 AND tenant_id=$2
		 ORDER BY timestamp DESC LIMIT 1`,
		deviceID, tenantID,
	)

	var ts time.Time
	var ignition, movement, sosEvent bool
	var extV, batV float64
	var batLvl, gnssStatus, gsmSig, rpm, fuel int
	var temp1 float64
	var ioData []byte

	err := row.Scan(&ts, &ignition, &movement, &extV, &batV, &batLvl,
		&gnssStatus, &gsmSig, &rpm, &fuel, &temp1, &sosEvent, &ioData)
	if err != nil {
		respondError(c, http.StatusNotFound, "no telemetry data found")
		return
	}

	respondOK(c, gin.H{
		"timestamp":        ts,
		"ignition":         ignition,
		"movement":         movement,
		"external_voltage": extV,
		"battery_voltage":  batV,
		"battery_level":    batLvl,
		"gnss_status":      gnssStatus,
		"gsm_signal":       gsmSig,
		"engine_rpm":       rpm,
		"fuel_level":       fuel,
		"temperature_1":    temp1,
		"sos_event":        sosEvent,
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseTimeRange(c *gin.Context) (time.Time, time.Time) {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	if s := c.Query("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = t
		}
	}
	if s := c.Query("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = t
		}
	}
	return from, to
}
