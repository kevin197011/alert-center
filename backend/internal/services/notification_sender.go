package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationSender sends alert notifications to bound channels.
type NotificationSender struct {
	db *pgxpool.Pool
}

// NewNotificationSender returns a new NotificationSender.
func NewNotificationSender(db *pgxpool.Pool) *NotificationSender {
	return &NotificationSender{db: db}
}

// SendToRuleChannels sends the alert payload to all channels bound to the rule.
func (s *NotificationSender) SendToRuleChannels(ctx context.Context, ruleID uuid.UUID, payload *AlertPayload) error {
	binding := &AlertChannelBindingService{db: s.db}
	return binding.SendToBoundChannels(ctx, ruleID, payload)
}
