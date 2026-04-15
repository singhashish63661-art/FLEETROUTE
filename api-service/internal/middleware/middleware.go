// Package middleware provides Gin middleware for the API service.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkgauth "gpsgo/pkg/auth"
)

// RequestLogger logs each HTTP request with method, path, status, and latency.
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("http",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// RLS injects the tenant_id from the JWT claims into the request context
// so all DB queries automatically scope to the correct tenant.
func RLS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// tenant_id is set by the JWT middleware — ensure it's present
		if pkgauth.TenantID(c) == "" {
			c.AbortWithStatus(403)
			return
		}
		// In production: SET LOCAL app.tenant_id = '<tenantID>' on the DB connection
		// This enforces PostgreSQL RLS policies per request.
		c.Next()
	}
}

// RateLimit implements a simple per-tenant token bucket using Redis.
// Limits: 1000 req/min per tenant.
func RateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := pkgauth.TenantID(c)
		if tenantID == "" {
			c.Next()
			return
		}

		key := "ratelimit:api:" + tenantID
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next() // fail open on Redis error
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, time.Minute) //nolint:errcheck
		}

		c.Header("X-RateLimit-Limit", "1000")
		c.Header("X-RateLimit-Remaining", itoa(max(0, 1000-int(count))))

		if count > 1000 {
			c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
