package services

import (
	"alert-center/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DataSourceService struct {
	db *pgxpool.Pool
}

func NewDataSourceService(db *pgxpool.Pool) *DataSourceService {
	return &DataSourceService{db: db}
}

func (s *DataSourceService) Create(ctx context.Context, req *CreateDataSourceRequest) (*models.DataSource, error) {
	config, _ := json.Marshal(req.Config)

	ds := &models.DataSource{
		ID:          uuid.New(),
		Name:        req.Name,
		Type:        req.Type,
		Description:  req.Description,
		Endpoint:    req.Endpoint,
		Config:      string(config),
		Status:      1,
		HealthStatus: "unknown",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO data_sources (id, name, type, description, endpoint, config, status, health_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, ds.ID, ds.Name, ds.Type, ds.Description, ds.Endpoint, ds.Config, ds.Status, ds.HealthStatus, ds.CreatedAt, ds.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func (s *DataSourceService) List(ctx context.Context, page, pageSize int, dataType string, status int) ([]models.DataSource, int, error) {
	offset := (page - 1) * pageSize

	rows, err := s.db.Query(ctx, `
		SELECT id, name, type, description, endpoint, config, status, health_status, last_check_at, created_at, updated_at
		FROM data_sources
		WHERE ($1 = '' OR type = $1) AND ($2 = -1 OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, dataType, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []models.DataSource
	for rows.Next() {
		var ds models.DataSource
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Description, &ds.Endpoint,
			&ds.Config, &ds.Status, &ds.HealthStatus, &ds.LastCheckAt, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, ds)
	}

	var total int
	s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM data_sources
		WHERE ($1 = '' OR type = $1) AND ($2 = -1 OR status = $2)
	`, dataType, status).Scan(&total)

	return list, total, nil
}

func (s *DataSourceService) GetByID(ctx context.Context, id uuid.UUID) (*models.DataSource, error) {
	var ds models.DataSource
	err := s.db.QueryRow(ctx, `
		SELECT id, name, type, description, endpoint, config, status, health_status, last_check_at, created_at, updated_at
		FROM data_sources WHERE id = $1
	`, id).Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Description, &ds.Endpoint,
		&ds.Config, &ds.Status, &ds.HealthStatus, &ds.LastCheckAt, &ds.CreatedAt, &ds.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (s *DataSourceService) HealthCheck(ctx context.Context, id uuid.UUID) error {
	var ds models.DataSource
	err := s.db.QueryRow(ctx, `
		SELECT id, name, type, endpoint, config FROM data_sources WHERE id = $1
	`, id).Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Endpoint, &ds.Config)
	if err != nil {
		return err
	}

	var healthy bool
	switch ds.Type {
	case "prometheus":
		healthy = checkPrometheusHealth(ctx, ds.Endpoint)
	case "victoria-metrics":
		healthy = checkVictoriaMetricsHealth(ctx, ds.Endpoint)
	default:
		healthy = true
	}

	healthStatus := "healthy"
	if !healthy {
		healthStatus = "unhealthy"
	}

	now := time.Now()
	s.db.Exec(ctx, `
		UPDATE data_sources SET health_status=$1, last_check_at=$2, updated_at=$2 WHERE id=$3
	`, healthStatus, now, id)

	return nil
}

func checkPrometheusHealth(ctx context.Context, endpoint string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	url := strings.TrimSuffix(endpoint, "/") + "/-/healthy"
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func checkVictoriaMetricsHealth(ctx context.Context, endpoint string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	url := strings.TrimSuffix(endpoint, "/") + "/health"
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (s *DataSourceService) Update(ctx context.Context, id uuid.UUID, req *UpdateDataSourceRequest) (*models.DataSource, error) {
	var ds models.DataSource
	err := s.db.QueryRow(ctx, `
		SELECT id, name, type, description, endpoint, config, status, health_status, created_at, updated_at
		FROM data_sources WHERE id = $1
	`, id).Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Description, &ds.Endpoint,
		&ds.Config, &ds.Status, &ds.HealthStatus, &ds.CreatedAt, &ds.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		ds.Name = *req.Name
	}
	if req.Description != nil {
		ds.Description = *req.Description
	}
	if req.Endpoint != nil {
		ds.Endpoint = *req.Endpoint
	}
	if req.Config != nil {
		config, _ := json.Marshal(req.Config)
		ds.Config = string(config)
	}
	if req.Status != nil {
		ds.Status = *req.Status
	}

	ds.UpdatedAt = time.Now()

	s.db.Exec(ctx, `
		UPDATE data_sources SET name=$1, description=$2, endpoint=$3, config=$4, status=$5, updated_at=$6 WHERE id=$7
	`, ds.Name, ds.Description, ds.Endpoint, ds.Config, ds.Status, ds.UpdatedAt, ds.ID)

	return &ds, nil
}

func (s *DataSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM data_sources WHERE id=$1`, id)
	return err
}

type CreateDataSourceRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Description string                 `json:"description"`
	Endpoint    string                 `json:"endpoint" binding:"required"`
	Config      map[string]interface{} `json:"config"`
}

type UpdateDataSourceRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Endpoint    *string                `json:"endpoint"`
	Config      *map[string]interface{} `json:"config"`
	Status     *int                  `json:"status"`
}
