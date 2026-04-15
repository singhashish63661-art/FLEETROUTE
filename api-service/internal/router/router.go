// Package router configures all API routes and middleware.
package router

import (
	"context"
	"net/http"
	"time"

	ada "github.com/GoAdminGroup/go-admin/adapter/gin"
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/postgres"
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/GoAdminGroup/themes/adminlte"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gpsgo/api-service/internal/adminpanel"
	"gpsgo/api-service/internal/handler"
	"gpsgo/api-service/internal/middleware"
	pkgauth "gpsgo/pkg/auth"
)

// New builds and returns the configured Gin engine.
func New(
	pool *pgxpool.Pool,
	rdb *redis.Client,
	authMgr *pkgauth.Manager,
	logger *zap.Logger,
	timescaleDSN string,
) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// ── Global middleware ─────────────────────────────────────────────────────
	r.Use(middleware.RequestLogger(logger))
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// ── Health endpoints (no auth) ────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api"})
	})
	r.GET("/ready", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// ── GoAdmin panel ───────────────────────────────────────────────────────────
	ensureGoAdminTables(pool, logger)
	adminEngine := engine.Default()
	adminCfg := config.Config{
		Databases: config.DatabaseList{
			"default": {
				Driver:          config.DriverPostgresql,
				Dsn:             timescaleDSN,
				MaxIdleConns:    10,
				MaxOpenConns:    50,
				ConnMaxLifetime: 5 * time.Minute,
			},
		},
		UrlPrefix: "admin",
		Store: config.Store{
			Path:   "./uploads",
			Prefix: "uploads",
		},
		Language:    language.EN,
		ColorScheme: adminlte.ColorschemeSkinBlack,
	}
	if err := adminEngine.AddConfig(&adminCfg).
		AddAuthService(adminpanel.BackendAuthProcessor(pool, logger)).
		AddAdapter(new(ada.Gin)).
		AddGenerators(adminpanel.Generators()).
		Use(r); err != nil {
		logger.Error("go-admin initialization failed", zap.Error(err))
	}
	r.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/admin/login")
	})
	r.GET("/admin/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/admin/login")
	})
	r.Static("/uploads", "./uploads")

	// ── Handlers ──────────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(pool, authMgr, logger)
	deviceHandler := handler.NewDeviceHandler(pool, rdb, logger)
	vehicleHandler := handler.NewVehicleHandler(pool, logger)
	geofenceHandler := handler.NewGeofenceHandler(pool, logger)
	alertHandler := handler.NewAlertHandler(pool, logger)
	ruleHandler := handler.NewRuleHandler(pool, logger)
	reportHandler := handler.NewReportHandler(pool, logger)
	driverHandler := handler.NewDriverHandler(pool, logger)
	settingsHandler := handler.NewSettingsHandler(pool, logger)

	// ── v1 routes ─────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")

	// Public auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// Protected routes — require valid JWT
	protected := v1.Group("")
	protected.Use(pkgauth.Middleware(authMgr))
	protected.Use(middleware.RLS())
	protected.Use(middleware.RateLimit(rdb))
	{
		// Devices
		devices := protected.Group("/devices")
		devices.GET("", deviceHandler.List)
		devices.POST("", deviceHandler.Create)
		devices.GET("/:id", deviceHandler.Get)
		devices.PUT("/:id", deviceHandler.Update)
		devices.DELETE("/:id", deviceHandler.Delete)
		devices.GET("/:id/live", deviceHandler.Live)
		devices.GET("/:id/history", deviceHandler.History)
		devices.GET("/:id/trips", deviceHandler.Trips)
		devices.GET("/:id/telemetry", deviceHandler.Telemetry)

		// Vehicles
		vehicles := protected.Group("/vehicles")
		vehicles.GET("", vehicleHandler.List)
		vehicles.POST("", vehicleHandler.Create)
		vehicles.GET("/:id", vehicleHandler.Get)
		vehicles.PUT("/:id", vehicleHandler.Update)
		vehicles.DELETE("/:id", vehicleHandler.Delete)
		vehicles.POST("/:id/command", vehicleHandler.SendCommand)

		// Drivers
		drivers := protected.Group("/drivers")
		drivers.GET("", driverHandler.List)
		drivers.POST("", driverHandler.Create)
		drivers.GET("/:id", driverHandler.Get)
		drivers.GET("/:id/score", driverHandler.Score)

		// Geofences
		geofences := protected.Group("/geofences")
		geofences.GET("", geofenceHandler.List)
		geofences.POST("", geofenceHandler.Create)
		geofences.GET("/:id", geofenceHandler.Get)
		geofences.PUT("/:id", geofenceHandler.Update)
		geofences.DELETE("/:id", geofenceHandler.Delete)
		geofences.GET("/:id/events", geofenceHandler.Events)

		// Alerts
		alerts := protected.Group("/alerts")
		alerts.GET("", alertHandler.List)
		alerts.POST("/:id/acknowledge", alertHandler.Acknowledge)

		// Rules
		rules := protected.Group("/rules")
		rules.GET("", ruleHandler.List)
		rules.POST("", ruleHandler.Create)
		rules.GET("/:id", ruleHandler.Get)
		rules.PUT("/:id", ruleHandler.Update)
		rules.DELETE("/:id", ruleHandler.Delete)
		rules.GET("/templates", ruleHandler.Templates)

		// Reports
		reports := protected.Group("/reports")
		reports.GET("/trips", reportHandler.Trips)
		reports.GET("/fuel", reportHandler.Fuel)
		reports.GET("/driver-behavior", reportHandler.DriverBehavior)
		reports.GET("/geofence-violations", reportHandler.GeofenceViolations)

		// Settings
		settings := protected.Group("/settings")
		settings.GET("/overview", settingsHandler.Overview)
		settings.GET("/users", settingsHandler.Users)
		settings.PUT("/preferences", settingsHandler.UpdatePreferences)
	}

	return r
}

func ensureGoAdminTables(pool *pgxpool.Pool, logger *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS goadmin_session (
			id SERIAL PRIMARY KEY,
			sid VARCHAR(50) NOT NULL,
			"values" VARCHAR(3000) NOT NULL,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_site (
			id SERIAL PRIMARY KEY,
			key VARCHAR(100) NOT NULL,
			value TEXT NOT NULL,
			type INTEGER DEFAULT 0,
			description VARCHAR(3000),
			state INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(190) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			avatar VARCHAR(255),
			remember_token VARCHAR(100),
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_roles (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_permissions (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL UNIQUE,
			http_method VARCHAR(255),
			http_path TEXT,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_menu (
			id SERIAL PRIMARY KEY,
			parent_id BIGINT DEFAULT 0,
			type INTEGER DEFAULT 1,
			"order" INTEGER DEFAULT 0,
			title VARCHAR(255) NOT NULL,
			icon VARCHAR(255),
			uri VARCHAR(255),
			header VARCHAR(255),
			plugin_name VARCHAR(150) NOT NULL DEFAULT '',
			uuid VARCHAR(150) NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_role_users (
			role_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now(),
			PRIMARY KEY (role_id, user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_role_permissions (
			role_id BIGINT NOT NULL,
			permission_id BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now(),
			PRIMARY KEY (role_id, permission_id)
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_role_menu (
			role_id BIGINT NOT NULL,
			menu_id BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now(),
			PRIMARY KEY (role_id, menu_id)
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_user_permissions (
			user_id BIGINT NOT NULL,
			permission_id BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now(),
			PRIMARY KEY (user_id, permission_id)
		);`,
		`CREATE TABLE IF NOT EXISTS goadmin_operation_log (
			id SERIAL PRIMARY KEY,
			user_id BIGINT,
			path VARCHAR(255),
			method VARCHAR(10),
			ip VARCHAR(50),
			input TEXT,
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now()
		);`,
		`INSERT INTO goadmin_roles (name, slug)
		 VALUES ('Administrator', 'administrator')
		 ON CONFLICT (slug) DO NOTHING;`,
		`INSERT INTO goadmin_permissions (name, slug, http_method, http_path)
		 VALUES ('All permission', '*', '', '*')
		 ON CONFLICT (slug) DO NOTHING;`,
		`INSERT INTO goadmin_menu (parent_id, type, "order", title, icon, uri, header, plugin_name, uuid)
		 VALUES (0, 1, 1, 'Dashboard', 'fa-bar-chart', '/', '', '', '')
		 ON CONFLICT DO NOTHING;`,
		`INSERT INTO goadmin_role_permissions (role_id, permission_id)
		 SELECT r.id, p.id
		 FROM goadmin_roles r, goadmin_permissions p
		 WHERE r.slug='administrator' AND p.slug='*'
		 ON CONFLICT DO NOTHING;`,
		`INSERT INTO goadmin_role_menu (role_id, menu_id)
		 SELECT r.id, m.id
		 FROM goadmin_roles r, goadmin_menu m
		 WHERE r.slug='administrator' AND m.uri='/'
		 ON CONFLICT DO NOTHING;`,
	}

	for _, stmt := range statements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			logger.Warn("failed to ensure go-admin bootstrap", zap.Error(err))
		}
	}
}
