package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BatchImportHandler struct {
	alertRuleService   *services.AlertRuleService
	alertSilenceService *services.AlertSilenceService
}

func NewBatchImportHandler(alertRuleService *services.AlertRuleService, alertSilenceService *services.AlertSilenceService) *BatchImportHandler {
	return &BatchImportHandler{
		alertRuleService:   alertRuleService,
		alertSilenceService: alertSilenceService,
	}
}

type ImportRequest struct {
	Rules []services.CreateAlertRuleRequest `json:"rules" binding:"required"`
}

type ImportResult struct {
	Success int      `json:"success"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors"`
}

func (h *BatchImportHandler) ImportRules(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result := &ImportResult{
		Success: 0,
		Failed:   0,
		Errors:   []string{},
	}

	for i, rule := range req.Rules {
		_, err := h.alertRuleService.Create(c.Request.Context(), &rule)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, "Rule "+strconv.Itoa(i)+": "+err.Error())
		} else {
			result.Success++
		}
	}

	response.Success(c, result)
}

type ExportRequest struct {
	GroupID  string `json:"group_id"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
}

func (h *BatchImportHandler) ExportRules(c *gin.Context) {
	var req ExportRequest
	c.ShouldBindQuery(&req)

	listReq := &services.ListAlertRuleRequest{
		Page:     1,
		PageSize: 10000,
		GroupID:  req.GroupID,
		Severity: req.Severity,
		Status:   req.Status,
	}

	rules, _, err := h.alertRuleService.List(c.Request.Context(), listReq)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	type ExportRule struct {
		Name            string   `json:"name"`
		Description    string   `json:"description"`
		Expression     string   `json:"expression"`
		ForDuration     int      `json:"for_duration"`
		Severity        string   `json:"severity"`
		Labels         string   `json:"labels"`
		Annotations    string   `json:"annotations"`
		GroupID        string   `json:"group_id"`
		DataSourceType string   `json:"data_source_type"`
		DataSourceURL  string   `json:"data_source_url"`
	}

	var exportRules []ExportRule
	for _, rule := range rules {
		exportRules = append(exportRules, ExportRule{
			Name:            rule.Name,
			Description:    rule.Description,
			Expression:     rule.Expression,
			ForDuration:     rule.ForDuration,
			Severity:        rule.Severity,
			Labels:         rule.Labels,
			Annotations:    rule.Annotations,
			GroupID:        rule.GroupID.String(),
			DataSourceType: rule.DataSourceType,
			DataSourceURL:  rule.DataSourceURL,
		})
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=alert_rules_export_"+time.Now().Format("20060102150405")+".json")
	c.JSON(http.StatusOK, exportRules)
}

type ExportChannelRequest struct {
	Type string `json:"type"`
}

func (h *BatchImportHandler) ExportChannels(c *gin.Context) {
	// Similar implementation for channels
	c.JSON(http.StatusOK, gin.H{"message": "export channels"})
}

type ImportSilenceRequest struct {
	Silences []services.CreateSilenceRequest `json:"silences" binding:"required"`
}

func (h *BatchImportHandler) ImportSilences(c *gin.Context) {
	var req ImportSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result := &ImportResult{
		Success: 0,
		Failed:   0,
		Errors:   []string{},
	}

	for i, silence := range req.Silences {
		_, err := h.alertSilenceService.Create(c.Request.Context(), &silence, uuid.Nil)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, "Silence "+strconv.Itoa(i)+": "+err.Error())
		} else {
			result.Success++
		}
	}

	response.Success(c, result)
}

func (h *BatchImportHandler) ExportSilences(c *gin.Context) {
	list, _, err := h.alertSilenceService.List(c.Request.Context(), 1, 10000, -1)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	type ExportSilence struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Matchers    []map[string]string `json:"matchers"`
		StartTime   time.Time          `json:"start_time"`
		EndTime     time.Time          `json:"end_time"`
	}

	var exportSilences []ExportSilence
	for _, silence := range list {
		var matchers []map[string]string
		json.Unmarshal([]byte(silence.Matchers), &matchers)
		exportSilences = append(exportSilences, ExportSilence{
			Name:        silence.Name,
			Description: silence.Description,
			Matchers:    matchers,
			StartTime:   silence.StartTime,
			EndTime:     silence.EndTime,
		})
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=alert_silences_export_"+time.Now().Format("20060102150405")+".json")
	c.JSON(http.StatusOK, exportSilences)
}
