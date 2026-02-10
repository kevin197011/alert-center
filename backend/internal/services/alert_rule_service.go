package services

import (
	"alert-center/internal/models"
	"alert-center/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// optionalUUID allows distinguishing "key absent" from "key present with null" in JSON for PATCH-style updates.
type optionalUUID struct {
	Value *uuid.UUID
	Set   bool
}

func (o *optionalUUID) UnmarshalJSON(data []byte) error {
	o.Set = true
	if len(data) == 4 && string(data) == "null" {
		o.Value = nil
		return nil
	}
	var u uuid.UUID
	if err := json.Unmarshal(data, &u); err != nil {
		return err
	}
	o.Value = &u
	return nil
}

type AlertRuleService struct {
	repo    *repository.AlertRuleRepository
	channel *repository.AlertChannelRepository
	history *repository.AlertHistoryRepository
}

func NewAlertRuleService(repo *repository.AlertRuleRepository,
	channel *repository.AlertChannelRepository,
	history *repository.AlertHistoryRepository) *AlertRuleService {
	return &AlertRuleService{repo: repo, channel: channel, history: history}
}

func (s *AlertRuleService) Create(ctx context.Context, req *CreateAlertRuleRequest) (*models.AlertRule, error) {
	labels, _ := json.Marshal(req.Labels)
	annotations, _ := json.Marshal(req.Annotations)

	effectiveStart := req.EffectiveStartTime
	if effectiveStart == "" {
		effectiveStart = "00:00"
	}
	effectiveEnd := req.EffectiveEndTime
	if effectiveEnd == "" {
		effectiveEnd = "23:59"
	}
	exclJSON := "[]"
	if len(req.ExclusionWindows) > 0 {
		b, _ := json.Marshal(req.ExclusionWindows)
		exclJSON = string(b)
	}
	evalInterval := req.EvaluationIntervalSeconds
	if evalInterval <= 0 {
		evalInterval = 60
	}
	status := req.Status
	if status != 0 && status != 1 {
		status = 1
	}
	rule := &models.AlertRule{
		Name:                       req.Name,
		Description:                req.Description,
		Expression:                 req.Expression,
		EvaluationIntervalSeconds:  evalInterval,
		ForDuration:                req.ForDuration,
		Severity:                   req.Severity,
		Labels:             string(labels),
		Annotations:        string(annotations),
		TemplateID:         req.TemplateID,
		GroupID:            req.GroupID,
		DataSourceType:     req.DataSourceType,
		DataSourceURL:      req.DataSourceURL,
		Status:             status,
		EffectiveStartTime: effectiveStart,
		EffectiveEndTime:   effectiveEnd,
		ExclusionWindows:   exclJSON,
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *AlertRuleService) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertRule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AlertRuleService) List(ctx context.Context, req *ListAlertRuleRequest) ([]models.AlertRule, int, error) {
	var groupID *uuid.UUID
	if req.GroupID != "" {
		gid, _ := uuid.Parse(req.GroupID)
		groupID = &gid
	}
	return s.repo.List(ctx, req.Page, req.PageSize, groupID, req.Severity, req.Status)
}

func (s *AlertRuleService) Update(ctx context.Context, id uuid.UUID, req *UpdateAlertRuleRequest) (*models.AlertRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Expression != nil {
		rule.Expression = *req.Expression
	}
	if req.EvaluationIntervalSeconds != nil {
		v := *req.EvaluationIntervalSeconds
		if v <= 0 {
			v = 60
		}
		rule.EvaluationIntervalSeconds = v
	}
	if req.ForDuration != nil {
		rule.ForDuration = *req.ForDuration
	}
	if req.Severity != nil {
		rule.Severity = *req.Severity
	}
	if req.Labels != nil {
		labels, _ := json.Marshal(req.Labels)
		rule.Labels = string(labels)
	}
	if req.Annotations != nil {
		annotations, _ := json.Marshal(req.Annotations)
		rule.Annotations = string(annotations)
	}
	if req.TemplateID.Set {
		rule.TemplateID = req.TemplateID.Value
	}
	if req.GroupID != nil {
		rule.GroupID = *req.GroupID
	}
	if req.DataSourceType != nil {
		rule.DataSourceType = *req.DataSourceType
	}
	if req.DataSourceURL != nil {
		rule.DataSourceURL = *req.DataSourceURL
	}
	if req.Status != nil {
		rule.Status = *req.Status
	}
	if req.EffectiveStartTime != nil {
		rule.EffectiveStartTime = *req.EffectiveStartTime
		if rule.EffectiveStartTime == "" {
			rule.EffectiveStartTime = "00:00"
		}
	}
	if req.EffectiveEndTime != nil {
		rule.EffectiveEndTime = *req.EffectiveEndTime
		if rule.EffectiveEndTime == "" {
			rule.EffectiveEndTime = "23:59"
		}
	}
	if req.ExclusionWindows != nil {
		exclJSON := "[]"
		if len(*req.ExclusionWindows) > 0 {
			b, _ := json.Marshal(*req.ExclusionWindows)
			exclJSON = string(b)
		}
		rule.ExclusionWindows = exclJSON
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *AlertRuleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *AlertRuleService) GetStatistics(ctx context.Context, req *StatisticsRequest) (map[string]interface{}, error) {
	var startTime, endTime *time.Time
	if req.StartTime != "" {
		t, _ := time.Parse("2006-01-02", req.StartTime)
		startTime = &t
	}
	if req.EndTime != "" {
		t, _ := time.Parse("2006-01-02", req.EndTime)
		endTime = &t
	}

	return s.history.GetStatistics(ctx, startTime, endTime, nil)
}

// PrometheusService handles Prometheus/VictoriaMetrics integration
type PrometheusService struct {
	client *http.Client
}

func NewPrometheusService() *PrometheusService {
	return &PrometheusService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *PrometheusService) Query(ctx context.Context, url, query string, queryTime time.Time) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/query", url), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("time", queryTime.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

type CreateAlertRuleRequest struct {
	Name                       string                  `json:"name" binding:"required"`
	Description                string                  `json:"description"`
	Expression                 string                  `json:"expression" binding:"required"`
	EvaluationIntervalSeconds  int                     `json:"evaluation_interval_seconds"` // 执行频率(秒), default 60
	ForDuration                int                     `json:"for_duration"`
	Severity                   string                  `json:"severity" binding:"required"`
	Labels             map[string]string       `json:"labels"`
	Annotations        map[string]string       `json:"annotations"`
	TemplateID         *uuid.UUID               `json:"template_id"`
	GroupID            uuid.UUID               `json:"group_id" binding:"required"`
	DataSourceType     string                  `json:"data_source_type"`
	DataSourceURL      string                  `json:"data_source_url"`
	EffectiveStartTime string                  `json:"effective_start_time"` // HH:MM, default 00:00
	EffectiveEndTime   string                  `json:"effective_end_time"`   // HH:MM, default 23:59
	ExclusionWindows   []models.ExclusionWindow `json:"exclusion_windows"`
	Status             int                     `json:"status"` // 0=禁用, 1=启用, default 1
}

type ListAlertRuleRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	GroupID  string `form:"group_id"`
	Severity string `form:"severity"`
	Status   string `form:"status"`
}

type UpdateAlertRuleRequest struct {
	Name                      *string            `json:"name"`
	Description               *string            `json:"description"`
	Expression                *string            `json:"expression"`
	EvaluationIntervalSeconds *int               `json:"evaluation_interval_seconds"`
	ForDuration               *int               `json:"for_duration"`
	Severity                  *string            `json:"severity"`
	Labels         *map[string]string `json:"labels"`
	Annotations    *map[string]string `json:"annotations"`
	TemplateID         optionalUUID              `json:"template_id"`
	GroupID            *uuid.UUID                `json:"group_id"`
	DataSourceType     *string                   `json:"data_source_type"`
	DataSourceURL      *string                   `json:"data_source_url"`
	Status             *int                      `json:"status"`
	EffectiveStartTime *string                   `json:"effective_start_time"`
	EffectiveEndTime   *string                   `json:"effective_end_time"`
	ExclusionWindows   *[]models.ExclusionWindow `json:"exclusion_windows"`
}

type StatisticsRequest struct {
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}
