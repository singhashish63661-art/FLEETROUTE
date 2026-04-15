package enrichment

import (
	"context"

	"go.uber.org/zap"
)

type NotificationManager struct {
	logger *zap.Logger
}

func NewNotificationManager(logger *zap.Logger) *NotificationManager {
	return &NotificationManager{logger: logger}
}

func (m *NotificationManager) Dispatch(ctx context.Context, tenantID, deviceID, message string) {
	m.logger.Info("Dispatching notification", 
		zap.String("tenant", tenantID), 
		zap.String("message", message))
}
