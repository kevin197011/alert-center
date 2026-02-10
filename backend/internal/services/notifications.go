package services

import "time"

// Broadcaster delivers real-time notifications (e.g. WebSocket).
// Implemented by handlers.WebSocketHandler to avoid package cycles.
type Broadcaster interface {
	SendAlertNotification(notification *AlertNotification)
	SendSLABreachNotification(notification *SLABreachNotification)
	SendTicketNotification(notification *TicketNotification)
}

type AlertNotification struct {
	AlertID   string            `json:"alert_id"`
	RuleID    string            `json:"rule_id"`
	RuleName  string            `json:"rule_name"`
	Severity  string            `json:"severity"`
	Status    string            `json:"status"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

type SLABreachNotification struct {
	BreachID  string    `json:"breach_id"`
	AlertID   string    `json:"alert_id"`
	Severity  string    `json:"severity"`
	BreachType string   `json:"breach_type"`
	Timestamp time.Time `json:"timestamp"`
}

type TicketNotification struct {
	TicketID  string    `json:"ticket_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}
