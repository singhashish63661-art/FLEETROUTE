// Package auth provides JWT RS256 token generation/validation and Gin middleware.
package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role represents a user's permission level in the RBAC hierarchy.
type Role string

const (
	RoleSuperAdmin   Role = "super_admin"
	RoleTenantAdmin  Role = "tenant_admin"
	RoleFleetManager Role = "fleet_manager"
	RoleDispatcher   Role = "dispatcher"
	RoleDriver       Role = "driver"
)

// Claims is the JWT payload.
type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

// Manager handles JWT signing and verification using RS256.
type Manager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager loads RSA keys from disk and returns a Manager.
func NewManager(privateKeyPath, publicKeyPath string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	privBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &Manager{
		privateKey: privKey,
		publicKey:  pubKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

// GenerateAccess returns a signed access token for the given user.
func (m *Manager) GenerateAccess(userID, tenantID string, role Role) (string, error) {
	return m.sign(userID, tenantID, role, m.accessTTL)
}

// GenerateRefresh returns a signed refresh token.
func (m *Manager) GenerateRefresh(userID, tenantID string, role Role) (string, error) {
	return m.sign(userID, tenantID, role, m.refreshTTL)
}

func (m *Manager) sign(userID, tenantID string, role Role, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "gpsgo",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// Validate parses and validates a token string, returning its claims.
func (m *Manager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// ── Permission matrix ─────────────────────────────────────────────────────────

// Permission represents a discrete action right.
type Permission string

const (
	PermReadDevices    Permission = "devices:read"
	PermWriteDevices   Permission = "devices:write"
	PermReadTrips      Permission = "trips:read"
	PermReadAlerts     Permission = "alerts:read"
	PermAckAlerts      Permission = "alerts:ack"
	PermWriteRules     Permission = "rules:write"
	PermReadReports    Permission = "reports:read"
	PermManageUsers    Permission = "users:manage"
	PermManageTenants  Permission = "tenants:manage"
	PermSendCommands   Permission = "commands:send"
)

// rolePermissions maps each role to its allowed permissions.
var rolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermReadDevices, PermWriteDevices, PermReadTrips, PermReadAlerts,
		PermAckAlerts, PermWriteRules, PermReadReports, PermManageUsers,
		PermManageTenants, PermSendCommands,
	},
	RoleTenantAdmin: {
		PermReadDevices, PermWriteDevices, PermReadTrips, PermReadAlerts,
		PermAckAlerts, PermWriteRules, PermReadReports, PermManageUsers, PermSendCommands,
	},
	RoleFleetManager: {
		PermReadDevices, PermReadTrips, PermReadAlerts, PermAckAlerts,
		PermWriteRules, PermReadReports, PermSendCommands,
	},
	RoleDispatcher: {
		PermReadDevices, PermReadTrips, PermReadAlerts, PermAckAlerts, PermReadReports,
	},
	RoleDriver: {
		PermReadDevices, PermReadTrips,
	},
}

// HasPermission returns true if the role is granted the given permission.
func HasPermission(role Role, perm Permission) bool {
	for _, p := range rolePermissions[role] {
		if p == perm {
			return true
		}
	}
	return false
}
