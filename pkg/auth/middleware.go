package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyTenantID = "tenant_id"
	ContextKeyRole     = "role"
)

// Middleware returns a Gin middleware handler that validates the Bearer JWT token.
func Middleware(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(raw, "Bearer ")
		claims, err := mgr.Validate(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyTenantID, claims.TenantID)
		c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}

// RequirePermission returns a Gin middleware that enforces a specific permission.
func RequirePermission(perm Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no role context"})
			return
		}
		role, ok := roleVal.(Role)
		if !ok || !HasPermission(role, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// TenantID extracts the tenant ID from a Gin context (set by Middleware).
func TenantID(c *gin.Context) string {
	v, _ := c.Get(ContextKeyTenantID)
	s, _ := v.(string)
	return s
}

// UserID extracts the user ID from a Gin context.
func UserID(c *gin.Context) string {
	v, _ := c.Get(ContextKeyUserID)
	s, _ := v.(string)
	return s
}
