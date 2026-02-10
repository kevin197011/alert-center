package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"alert-center/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertTemplateService struct {
	db *pgxpool.Pool
}

func NewAlertTemplateService(db *pgxpool.Pool) *AlertTemplateService {
	return &AlertTemplateService{db: db}
}

func (s *AlertTemplateService) Create(ctx context.Context, req *CreateTemplateRequest) (*models.AlertTemplate, error) {
	variablesJSON := "{}"
	if req.Variables != nil {
		b, _ := json.Marshal(req.Variables)
		variablesJSON = string(b)
		if variablesJSON == "null" {
			variablesJSON = "{}"
		}
	}
	templateType := req.Type
	if templateType == "" {
		templateType = "markdown"
	}
	template := &models.AlertTemplate{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Variables:   variablesJSON,
		Type:        templateType,
		GroupID:     req.GroupID,
		Status:      1,
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO alert_templates (id, name, description, content, variables, type, group_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`, template.ID, template.Name, template.Description, template.Content, template.Variables,
		template.Type, template.GroupID, template.Status)
	if err != nil {
		return nil, err
	}

	return template, nil
}

func (s *AlertTemplateService) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertTemplate, error) {
	var template models.AlertTemplate
	err := s.db.QueryRow(ctx, `
		SELECT id, name, description, content, variables, type, group_id, status, created_at, updated_at
		FROM alert_templates WHERE id = $1
	`, id).Scan(&template.ID, &template.Name, &template.Description, &template.Content,
		&template.Variables, &template.Type, &template.GroupID, &template.Status,
		&template.CreatedAt, &template.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (s *AlertTemplateService) List(ctx context.Context, page, pageSize int, templateType string, status int) ([]models.AlertTemplate, int, error) {
	offset := (page - 1) * pageSize

	rows, err := s.db.Query(ctx, `
		SELECT id, name, description, content, variables, type, group_id, status, created_at, updated_at
		FROM alert_templates
		WHERE ($1 = '' OR type = $1) AND ($2 = -1 OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, templateType, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []models.AlertTemplate
	for rows.Next() {
		var t models.AlertTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Content,
			&t.Variables, &t.Type, &t.GroupID, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		templates = append(templates, t)
	}

	var total int
	s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM alert_templates
		WHERE ($1 = '' OR type = $1) AND ($2 = -1 OR status = $2)
	`, templateType, status).Scan(&total)

	return templates, total, nil
}

func (s *AlertTemplateService) Update(ctx context.Context, id uuid.UUID, req *UpdateTemplateRequest) (*models.AlertTemplate, error) {
	template, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.Content != nil {
		template.Content = *req.Content
	}
	if req.Variables != nil {
		variables, _ := json.Marshal(req.Variables)
		template.Variables = string(variables)
	}
	if req.Type != nil {
		template.Type = *req.Type
	}

	_, err = s.db.Exec(ctx, `
		UPDATE alert_templates SET name=$1, description=$2, content=$3, variables=$4, type=$5, updated_at=NOW()
		WHERE id=$6
	`, template.Name, template.Description, template.Content, template.Variables, template.Type, template.ID)
	if err != nil {
		return nil, err
	}

	return template, nil
}

func (s *AlertTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE alert_templates SET status=0 WHERE id=$1`, id)
	return err
}

func (s *AlertTemplateService) Render(ctx context.Context, templateID uuid.UUID, data map[string]interface{}) (string, error) {
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return "", err
	}

	content := template.Content
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		content = strings.ReplaceAll(content, placeholder, fmt.Sprintf("%v", value))
	}

	return content, nil
}

type CreateTemplateRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Content     string                 `json:"content" binding:"required"`
	Variables   map[string]string     `json:"variables"`
	Type        string                 `json:"type"`
	GroupID     *uuid.UUID            `json:"group_id"`
}

type UpdateTemplateRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Content     *string                `json:"content"`
	Variables   *map[string]string     `json:"variables"`
	Type        *string                `json:"type"`
}
