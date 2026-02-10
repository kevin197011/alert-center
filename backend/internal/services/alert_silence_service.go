package services

import (
	"alert-center/internal/models"
	"context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertSilenceService struct {
	db *pgxpool.Pool
}

func NewAlertSilenceService(db *pgxpool.Pool) *AlertSilenceService {
	return &AlertSilenceService{db: db}
}

func (s *AlertSilenceService) Create(ctx context.Context, req *CreateSilenceRequest, userID uuid.UUID) (*models.AlertSilence, error) {
	matchers, _ := json.Marshal(req.Matchers)

	silence := &models.AlertSilence{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Matchers:    string(matchers),
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		CreatedBy:   userID,
		Status:      1,
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO alert_silences (id, name, description, matchers, start_time, end_time, created_by, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, silence.ID, silence.Name, silence.Description, silence.Matchers,
		silence.StartTime, silence.EndTime, silence.CreatedBy, silence.Status, time.Now(), time.Now())
	if err != nil {
		return nil, err
	}

	return silence, nil
}

func (s *AlertSilenceService) List(ctx context.Context, page, pageSize int, status int) ([]models.AlertSilence, int, error) {
	offset := (page - 1) * pageSize

	rows, err := s.db.Query(ctx, `
		SELECT id, name, description, matchers, start_time, end_time, created_by, status, created_at, updated_at
		FROM alert_silences
		WHERE status = $1 OR $2 = -1
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, status, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []models.AlertSilence
	for rows.Next() {
		var silence models.AlertSilence
		if err := rows.Scan(&silence.ID, &silence.Name, &silence.Description, &silence.Matchers,
			&silence.StartTime, &silence.EndTime, &silence.CreatedBy,
			&silence.Status, &silence.CreatedAt, &silence.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, silence)
	}

	var total int
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_silences WHERE status = $1 OR $2 = -1`, status, status).Scan(&total)

	return list, total, nil
}

func (s *AlertSilenceService) IsSilenced(ctx context.Context, labels map[string]string) (bool, error) {
	now := time.Now()
	
	rows, err := s.db.Query(ctx, `
		SELECT id, matchers FROM alert_silences
		WHERE status = 1 AND start_time <= $1 AND end_time >= $1
	`, now)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var matchers string
		rows.Scan(&id, &matchers)

		var silenceMatchers []map[string]string
		json.Unmarshal([]byte(matchers), &silenceMatchers)

		for _, sm := range silenceMatchers {
			match := true
			for key, pattern := range sm {
				labelValue, exists := labels[key]
				if !exists {
					match = false
					break
				}
				
				if len(pattern) >= 2 && pattern[0:2] == "~" {
					regexPattern := pattern[2:]
					re, err := regexp.Compile("^" + regexPattern + "$")
					if err != nil {
						match = false
						break
					}
					if !re.MatchString(labelValue) {
						match = false
						break
					}
				} else {
					if labelValue != pattern {
						match = false
						break
					}
				}
			}
			if match {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *AlertSilenceService) Update(ctx context.Context, id uuid.UUID, req *UpdateSilenceRequest) (*models.AlertSilence, error) {
	silence, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		silence.Name = *req.Name
	}
	if req.Description != nil {
		silence.Description = *req.Description
	}
	if req.Matchers != nil {
		matchers, _ := json.Marshal(req.Matchers)
		silence.Matchers = string(matchers)
	}
	if req.StartTime != nil {
		silence.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		silence.EndTime = *req.EndTime
	}

	silence.UpdatedAt = time.Now()

	s.db.Exec(ctx, `
		UPDATE alert_silences SET name=$1, description=$2, matchers=$3, start_time=$4, end_time=$5, updated_at=$6
		WHERE id=$7
	`, silence.Name, silence.Description, silence.Matchers, silence.StartTime, silence.EndTime, silence.UpdatedAt, id)

	return silence, nil
}

func (s *AlertSilenceService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM alert_silences WHERE id=$1`, id)
	return err
}

func (s *AlertSilenceService) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertSilence, error) {
	var silence models.AlertSilence
	err := s.db.QueryRow(ctx, `
		SELECT id, name, description, matchers, start_time, end_time, created_by, status, created_at, updated_at
		FROM alert_silences WHERE id=$1
	`, id).Scan(&silence.ID, &silence.Name, &silence.Description, &silence.Matchers,
		&silence.StartTime, &silence.EndTime, &silence.CreatedBy,
		&silence.Status, &silence.CreatedAt, &silence.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &silence, nil
}

type CreateSilenceRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Matchers    []map[string]string `json:"matchers" binding:"required"`
	StartTime   time.Time        `json:"start_time" binding:"required"`
	EndTime     time.Time        `json:"end_time" binding:"required"`
}

type UpdateSilenceRequest struct {
	Name        *string            `json:"name"`
	Description *string            `json:"description"`
	Matchers    *[]map[string]string `json:"matchers"`
	StartTime   *time.Time        `json:"start_time"`
	EndTime     *time.Time        `json:"end_time"`
}
