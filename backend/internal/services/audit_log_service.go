package services

import (
	"context"
	"encoding/json"
	"time"

	"alert-center/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogService struct {
	db *pgxpool.Pool
}

func NewAuditLogService(db *pgxpool.Pool) *AuditLogService {
	return &AuditLogService{db: db}
}

func (s *AuditLogService) Create(ctx context.Context, log *models.OperationLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	_, err := s.db.Exec(ctx, `
		INSERT INTO operation_logs (id, user_id, action, resource, resource_id, detail, ip, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, log.ID, log.UserID, log.Action, log.Resource, log.ResourceID, log.Detail, log.IP, log.CreatedAt)
	return err
}

func (s *AuditLogService) CreateWithDetail(ctx context.Context, userID uuid.UUID, action, resource, resourceID string, detail map[string]interface{}) error {
	detailJSON, _ := json.Marshal(detail)

	log := &models.OperationLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     string(detailJSON),
		CreatedAt:  time.Now(),
	}

	return s.Create(ctx, log)
}

func (s *AuditLogService) List(ctx context.Context, page, pageSize int, req *ListAuditLogRequest) ([]models.OperationLog, int, error) {
	offset := (page - 1) * pageSize
	startArg := req.StartTime
	if startArg == nil {
		t := time.Time{}
		startArg = &t
	}
	endArg := req.EndTime
	if endArg == nil {
		t := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
		endArg = &t
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, action, resource, resource_id, detail, ip, created_at
		FROM operation_logs
		WHERE ($1::uuid IS NULL OR user_id = $1)
			AND ($2 = '' OR action = $2)
			AND ($3 = '' OR resource = $3)
			AND (created_at >= $4 AND created_at <= $5)
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7
	`, req.UserID, req.Action, req.Resource, startArg, endArg, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []models.OperationLog
	for rows.Next() {
		var log models.OperationLog
		if err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.Resource, &log.ResourceID, &log.Detail, &log.IP, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	var total int
	s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM operation_logs
		WHERE ($1::uuid IS NULL OR user_id = $1)
			AND ($2 = '' OR action = $2)
			AND ($3 = '' OR resource = $3)
			AND (created_at >= $4 AND created_at <= $5)
	`, req.UserID, req.Action, req.Resource, startArg, endArg).Scan(&total)

	return logs, total, nil
}

func (s *AuditLogService) Export(ctx context.Context, req *ListAuditLogRequest) ([]models.OperationLog, error) {
	startArg := req.StartTime
	if startArg == nil {
		t := time.Time{}
		startArg = &t
	}
	endArg := req.EndTime
	if endArg == nil {
		t := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
		endArg = &t
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, action, resource, resource_id, detail, ip, created_at
		FROM operation_logs
		WHERE ($1::uuid IS NULL OR user_id = $1)
			AND ($2 = '' OR action = $2)
			AND ($3 = '' OR resource = $3)
			AND (created_at >= $4 AND created_at <= $5)
		ORDER BY created_at DESC
	`, req.UserID, req.Action, req.Resource, startArg, endArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.OperationLog
	for rows.Next() {
		var log models.OperationLog
		if err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.Resource, &log.ResourceID, &log.Detail, &log.IP, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

type ListAuditLogRequest struct {
	UserID    *uuid.UUID `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionLogin    = "login"
	ActionLogout   = "logout"
	ActionBind     = "bind"
	ActionUnbind   = "unbind"
	ActionExport   = "export"
)

const (
	ResourceUser          = "user"
	ResourceUserGroup     = "user_group"
	ResourceAlertRule     = "alert_rule"
	ResourceAlertChannel  = "alert_channel"
	ResourceAlertTemplate = "alert_template"
	ResourceAlertHistory  = "alert_history"
	ResourceBinding       = "binding"
)
