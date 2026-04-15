package notifications

import (
	"context"
	
	"go.uber.org/zap"
)

type Provider interface {
	Send(ctx context.Context, to string, message string) error
}

type NotificationManager struct {
	logger *zap.Logger
}

func NewNotificationManager(logger *zap.Logger) *NotificationManager {
	return &NotificationManager{logger: logger}
}

func (m *NotificationManager) Dispatch(ctx context.Context, tenantID, deviceID, message string) {
	// Stub for Twilio / AWS SNS / SendGrid integrations.
	// Normally this loads the Tenant's notification settings and dispatches HTTP requests.
	m.logger.Info("Dispatching notification", 
		zap.String("tenant", tenantID), 
		zap.String("message", message))
}
