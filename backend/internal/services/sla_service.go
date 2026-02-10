package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SLAService manages SLA configs and seed data.
type SLAService struct {
	db *pgxpool.Pool
}

// NewSLAService returns a new SLAService.
func NewSLAService(db *pgxpool.Pool) *SLAService {
	return &SLAService{db: db}
}

// SeedDefaultSLAConfigs inserts default severity-based SLA configs if none exist.
func (s *SLAService) SeedDefaultSLAConfigs(ctx context.Context) error {
	var count int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM sla_configs`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	defaults := []struct {
		name                 string
		severity             string
		responseMins, resolveMins int
		priority             int
	}{
		{"Critical SLA", "critical", 15, 60, 100},
		{"Warning SLA", "warning", 30, 120, 50},
		{"Info SLA", "info", 60, 240, 10},
	}
	for _, d := range defaults {
		id := uuid.New()
		now := time.Now()
		_, err := s.db.Exec(ctx, `
			INSERT INTO sla_configs (id, name, severity, response_time_mins, resolution_time_mins, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, d.name, d.severity, d.responseMins, d.resolveMins, d.priority, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTopConfigBySeverity returns the highest-priority SLA config for the given severity.
func (s *SLAService) GetTopConfigBySeverity(ctx context.Context, severity string) (uuid.UUID, int, int, error) {
	var id uuid.UUID
	var responseMins, resolutionMins int
	err := s.db.QueryRow(ctx, `
		SELECT id, response_time_mins, resolution_time_mins
		FROM sla_configs WHERE severity = $1
		ORDER BY priority DESC LIMIT 1
	`, severity).Scan(&id, &responseMins, &resolutionMins)
	if err != nil {
		return uuid.Nil, 0, 0, fmt.Errorf("sla config not found for severity %s", severity)
	}
	return id, responseMins, resolutionMins, nil
}

// CreateAlertSLA inserts per-alert SLA deadlines using the highest-priority config.
func (s *SLAService) CreateAlertSLA(ctx context.Context, alertID, ruleID uuid.UUID, severity string, startedAt time.Time) error {
	configID, responseMins, resolutionMins, err := s.GetTopConfigBySeverity(ctx, severity)
	if err != nil {
		return err
	}
	var exists int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_slas WHERE alert_id=$1`, alertID).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	slaID := uuid.New()
	responseDeadline := startedAt.Add(time.Duration(responseMins) * time.Minute)
	resolutionDeadline := startedAt.Add(time.Duration(resolutionMins) * time.Minute)
	_, err = s.db.Exec(ctx, `
		INSERT INTO alert_slas (id, alert_id, rule_id, severity, sla_config_id, response_deadline, resolution_deadline, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', NOW())
	`, slaID, alertID, ruleID, severity, configID, responseDeadline, resolutionDeadline)
	return err
}

// MarkResolved updates SLA record when alert is resolved.
func (s *SLAService) MarkResolved(ctx context.Context, alertID uuid.UUID, resolvedAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE alert_slas SET resolved_at=$1, status='resolved',
		resolution_time_secs=EXTRACT(EPOCH FROM ($1 - created_at))
		WHERE alert_id=$2
	`, resolvedAt, alertID)
	return err
}
