package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SLABreachService manages SLA breach records and notifications.
type SLABreachService struct {
	db     *pgxpool.Pool
	sender *NotificationSender
	broadcaster Broadcaster
}

// NewSLABreachService returns a new SLABreachService.
func NewSLABreachService(db *pgxpool.Pool, sender *NotificationSender, broadcaster Broadcaster) *SLABreachService {
	return &SLABreachService{db: db, sender: sender, broadcaster: broadcaster}
}

// SLABreach represents a breach record.
type SLABreach struct {
	ID           uuid.UUID  `json:"id"`
	AlertID      uuid.UUID  `json:"alert_id"`
	RuleID       uuid.UUID  `json:"rule_id"`
	Severity     string     `json:"severity"`
	BreachType   string     `json:"breach_type"`
	BreachTime   time.Time  `json:"breach_time"`
	ResponseTime float64    `json:"response_time"`
	AssignedTo   *uuid.UUID `json:"assigned_to"`
	AssignedName *string    `json:"assigned_name"`
	Notified     bool       `json:"notified"`
	CreatedAt    time.Time  `json:"created_at"`
}

// GetBreaches returns paginated breach list.
func (s *SLABreachService) GetBreaches(ctx context.Context, page, pageSize int, status string) ([]SLABreach, int, error) {
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	var total int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM sla_breaches`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, alert_id, rule_id, severity, breach_type, breach_time, response_time, assigned_to, assigned_name, notified, created_at
		FROM sla_breaches ORDER BY breach_time DESC LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []SLABreach
	for rows.Next() {
		var b SLABreach
		if err := rows.Scan(&b.ID, &b.AlertID, &b.RuleID, &b.Severity, &b.BreachType, &b.BreachTime, &b.ResponseTime, &b.AssignedTo, &b.AssignedName, &b.Notified, &b.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, b)
	}
	return list, total, nil
}

// GetBreachStats returns stats for a time range.
func (s *SLABreachService) GetBreachStats(ctx context.Context, startTime, endTime *time.Time) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"total_breaches": 0,
		"period_start":   nil,
		"period_end":     nil,
	}
	if startTime != nil {
		stats["period_start"] = startTime.Format(time.RFC3339)
	}
	if endTime != nil {
		stats["period_end"] = endTime.Format(time.RFC3339)
	}
	var total int
	if startTime != nil && endTime != nil {
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM sla_breaches WHERE breach_time >= $1 AND breach_time <= $2`, *startTime, *endTime).Scan(&total); err != nil {
			return nil, err
		}
	} else if startTime != nil {
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM sla_breaches WHERE breach_time >= $1`, *startTime).Scan(&total); err != nil {
			return nil, err
		}
	} else if endTime != nil {
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM sla_breaches WHERE breach_time <= $1`, *endTime).Scan(&total); err != nil {
			return nil, err
		}
	} else {
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM sla_breaches`).Scan(&total); err != nil {
			return nil, err
		}
	}
	stats["total_breaches"] = total
	return stats, nil
}

// TriggerCheck creates breach records for alerts that exceeded SLA (stub: no-op or minimal).
func (s *SLABreachService) TriggerCheck(ctx context.Context) (int, error) {
	now := time.Now()
	created := 0

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	// Response breaches: deadline passed, not breached yet, not acknowledged.
	rows, err := tx.Query(ctx, `
		SELECT alert_id, rule_id, severity, response_deadline, created_at
		FROM alert_slas
		WHERE response_deadline IS NOT NULL
		  AND response_deadline <= $1
		  AND response_breached = false
		  AND first_acked_at IS NULL
	`, now)
	if err != nil {
		return 0, err
	}
	type breachRow struct {
		alertID   uuid.UUID
		ruleID    uuid.UUID
		severity  string
		breachAt  time.Time
		createdAt time.Time
	}
	var responseRows []breachRow
	for rows.Next() {
		var r breachRow
		if err := rows.Scan(&r.alertID, &r.ruleID, &r.severity, &r.breachAt, &r.createdAt); err != nil {
			rows.Close()
			return 0, err
		}
		responseRows = append(responseRows, r)
	}
	rows.Close()
	for _, r := range responseRows {
		id := uuid.New()
		responseSecs := now.Sub(r.createdAt).Seconds()
		_, err = tx.Exec(ctx, `
			INSERT INTO sla_breaches (id, alert_id, rule_id, severity, breach_type, breach_time, response_time, notified, created_at)
			VALUES ($1, $2, $3, $4, 'response', $5, $6, false, NOW())
		`, id, r.alertID, r.ruleID, r.severity, r.breachAt, responseSecs)
		if err != nil {
			return 0, err
		}
		_, _ = tx.Exec(ctx, `UPDATE alert_slas SET response_breached=true WHERE alert_id=$1`, r.alertID)
		created++
		if s.broadcaster != nil {
			s.broadcaster.SendSLABreachNotification(&SLABreachNotification{
				BreachID:  id.String(),
				AlertID:   r.alertID.String(),
				Severity:  r.severity,
				BreachType: "response",
				Timestamp: now,
			})
		}
	}

	// Resolution breaches: deadline passed, not breached yet, not resolved.
	rows, err = tx.Query(ctx, `
		SELECT alert_id, rule_id, severity, resolution_deadline, created_at
		FROM alert_slas
		WHERE resolution_deadline IS NOT NULL
		  AND resolution_deadline <= $1
		  AND resolution_breached = false
		  AND resolved_at IS NULL
	`, now)
	if err != nil {
		return 0, err
	}
	var resolutionRows []breachRow
	for rows.Next() {
		var r breachRow
		if err := rows.Scan(&r.alertID, &r.ruleID, &r.severity, &r.breachAt, &r.createdAt); err != nil {
			rows.Close()
			return 0, err
		}
		resolutionRows = append(resolutionRows, r)
	}
	rows.Close()
	for _, r := range resolutionRows {
		id := uuid.New()
		responseSecs := now.Sub(r.createdAt).Seconds()
		_, err = tx.Exec(ctx, `
			INSERT INTO sla_breaches (id, alert_id, rule_id, severity, breach_type, breach_time, response_time, notified, created_at)
			VALUES ($1, $2, $3, $4, 'resolution', $5, $6, false, NOW())
		`, id, r.alertID, r.ruleID, r.severity, r.breachAt, responseSecs)
		if err != nil {
			return 0, err
		}
		_, _ = tx.Exec(ctx, `UPDATE alert_slas SET resolution_breached=true WHERE alert_id=$1`, r.alertID)
		created++
		if s.broadcaster != nil {
			s.broadcaster.SendSLABreachNotification(&SLABreachNotification{
				BreachID:  id.String(),
				AlertID:   r.alertID.String(),
				Severity:  r.severity,
				BreachType: "resolution",
				Timestamp: now,
			})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return created, nil
}

// TriggerNotifications sends notifications for unnotified breaches (stub).
func (s *SLABreachService) TriggerNotifications(ctx context.Context) (int, error) {
	// For now, just mark unnotified breaches as notified.
	rows, err := s.db.Query(ctx, `
		SELECT id, alert_id, severity, breach_type
		FROM sla_breaches WHERE notified=false
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, alertID uuid.UUID
		var severity, breachType string
		if err := rows.Scan(&id, &alertID, &severity, &breachType); err != nil {
			return 0, err
		}
		_, err := s.db.Exec(ctx, `UPDATE sla_breaches SET notified=true WHERE id=$1`, id)
		if err != nil {
			return 0, err
		}
		count++
		if s.broadcaster != nil {
			s.broadcaster.SendSLABreachNotification(&SLABreachNotification{
				BreachID:  id.String(),
				AlertID:   alertID.String(),
				Severity:  severity,
				BreachType: breachType,
				Timestamp: time.Now(),
			})
		}
	}
	return count, nil
}
