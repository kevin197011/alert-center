package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertEscalationService struct {
	db *pgxpool.Pool
}

func NewAlertEscalationMgmtService(db *pgxpool.Pool) *AlertEscalationService {
	return &AlertEscalationService{db: db}
}

type AlertEscalation struct {
	ID           uuid.UUID  `json:"id"`
	AlertID      uuid.UUID  `json:"alert_id"`
	FromUserID   uuid.UUID  `json:"from_user_id"`
	FromUsername string     `json:"from_username"`
	ToUserID     uuid.UUID  `json:"to_user_id"`
	ToUsername   string     `json:"to_username"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

type CreateEscalationRequest struct {
	AlertID    uuid.UUID `json:"alert_id" binding:"required"`
	ToUserID   uuid.UUID `json:"to_user_id" binding:"required"`
	ToUsername string    `json:"to_username" binding:"required"`
	Reason     string    `json:"reason" binding:"required"`
}

func (s *AlertEscalationService) CreateEscalation(ctx context.Context, fromUserID uuid.UUID, fromUsername string, req *CreateEscalationRequest) (*AlertEscalation, error) {
	esc := &AlertEscalation{
		ID:           uuid.New(),
		AlertID:      req.AlertID,
		FromUserID:   fromUserID,
		FromUsername: fromUsername,
		ToUserID:     req.ToUserID,
		ToUsername:   req.ToUsername,
		Reason:       req.Reason,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO user_escalations (id, alert_id, from_user_id, from_username, to_user_id, to_username, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, esc.ID, esc.AlertID, esc.FromUserID, esc.FromUsername, esc.ToUserID, esc.ToUsername, esc.Reason, esc.Status, esc.CreatedAt)
	if err != nil {
		return nil, err
	}
	log.Printf("Alert %s escalated from %s to %s", esc.AlertID, esc.FromUsername, esc.ToUsername)
	return esc, nil
}

func (s *AlertEscalationService) GetAlertEscalations(ctx context.Context, alertID uuid.UUID) ([]AlertEscalation, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, alert_id, from_user_id, from_username, to_user_id, to_username, reason, status, created_at, resolved_at
		FROM user_escalations WHERE alert_id = $1 ORDER BY created_at DESC
	`, alertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AlertEscalation
	for rows.Next() {
		var e AlertEscalation
		if err := rows.Scan(&e.ID, &e.AlertID, &e.FromUserID, &e.FromUsername, &e.ToUserID, &e.ToUsername, &e.Reason, &e.Status, &e.CreatedAt, &e.ResolvedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

func (s *AlertEscalationService) GetPendingEscalations(ctx context.Context, userID uuid.UUID) ([]AlertEscalation, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, alert_id, from_user_id, from_username, to_user_id, to_username, reason, status, created_at, resolved_at
		FROM user_escalations WHERE to_user_id = $1 AND status = 'pending' ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AlertEscalation
	for rows.Next() {
		var e AlertEscalation
		if err := rows.Scan(&e.ID, &e.AlertID, &e.FromUserID, &e.FromUsername, &e.ToUserID, &e.ToUsername, &e.Reason, &e.Status, &e.CreatedAt, &e.ResolvedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

func (s *AlertEscalationService) AcceptEscalation(ctx context.Context, escalationID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE user_escalations SET status='accepted', resolved_at=$1 WHERE id=$2 AND status='pending'`, now, escalationID)
	return err
}

func (s *AlertEscalationService) RejectEscalation(ctx context.Context, escalationID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE user_escalations SET status='rejected', resolved_at=$1 WHERE id=$2 AND status='pending'`, now, escalationID)
	return err
}

func (s *AlertEscalationService) ResolveEscalation(ctx context.Context, escalationID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE user_escalations SET status='resolved', resolved_at=$1 WHERE id=$2`, now, escalationID)
	return err
}
