package handlers

import (
	"alert-center/internal/repository"
	"alert-center/pkg/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SLAHandler handles SLA config and report APIs.
type SLAHandler struct {
	slaConfigRepo *repository.SLAConfigRepository
	slaRepo       *repository.AlertSLARepository
}

// NewSLAHandler returns a new SLAHandler.
func NewSLAHandler(repo *repository.SLAConfigRepository) *SLAHandler {
	return &SLAHandler{slaConfigRepo: repo}
}

// WithAlertSLARepository sets the alert SLA repository.
func (h *SLAHandler) WithAlertSLARepository(repo *repository.AlertSLARepository) *SLAHandler {
	h.slaRepo = repo
	return h
}

func (h *SLAHandler) ListSLAConfigs(c *gin.Context) {
	list, err := h.slaConfigRepo.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list, "total": len(list)})
}

func (h *SLAHandler) CreateSLAConfig(c *gin.Context) {
	var req struct {
		Name               string `json:"name" binding:"required"`
		Severity           string `json:"severity" binding:"required"`
		ResponseTimeMins   int    `json:"response_time_mins" binding:"required"`
		ResolutionTimeMins int   `json:"resolution_time_mins" binding:"required"`
		Priority           int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	config := &repository.SLAConfig{
		Name:               req.Name,
		Severity:           req.Severity,
		ResponseTimeMins:   req.ResponseTimeMins,
		ResolutionTimeMins: req.ResolutionTimeMins,
		Priority:           req.Priority,
	}
	if err := h.slaConfigRepo.Create(c.Request.Context(), config); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, config)
}

func (h *SLAHandler) GetSLAConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	config, err := h.slaConfigRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "config not found")
		return
	}
	response.Success(c, config)
}

func (h *SLAHandler) UpdateSLAConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	config, err := h.slaConfigRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "config not found")
		return
	}
	var req struct {
		Name               *string `json:"name"`
		Severity           *string `json:"severity"`
		ResponseTimeMins   *int    `json:"response_time_mins"`
		ResolutionTimeMins *int    `json:"resolution_time_mins"`
		Priority           *int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.Severity != nil {
		config.Severity = *req.Severity
	}
	if req.ResponseTimeMins != nil {
		config.ResponseTimeMins = *req.ResponseTimeMins
	}
	if req.ResolutionTimeMins != nil {
		config.ResolutionTimeMins = *req.ResolutionTimeMins
	}
	if req.Priority != nil {
		config.Priority = *req.Priority
	}
	if err := h.slaConfigRepo.Update(c.Request.Context(), config); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, config)
}

func (h *SLAHandler) DeleteSLAConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.slaConfigRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *SLAHandler) SeedDefaultSLAConfigs(c *gin.Context) {
	list, err := h.slaConfigRepo.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if len(list) > 0 {
		response.Success(c, gin.H{"message": "configs already exist"})
		return
	}
	defaults := []repository.SLAConfig{
		{Name: "Critical SLA", Severity: "critical", ResponseTimeMins: 15, ResolutionTimeMins: 60, Priority: 100},
		{Name: "Warning SLA", Severity: "warning", ResponseTimeMins: 30, ResolutionTimeMins: 120, Priority: 50},
		{Name: "Info SLA", Severity: "info", ResponseTimeMins: 60, ResolutionTimeMins: 240, Priority: 10},
	}
	for i := range defaults {
		if err := h.slaConfigRepo.Create(c.Request.Context(), &defaults[i]); err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	response.Success(c, gin.H{"message": "seeded default SLA configs"})
}

func (h *SLAHandler) GetAlertSLA(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("alert_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid alert_id")
		return
	}
	if h.slaRepo == nil {
		response.Error(c, http.StatusInternalServerError, "sla repository not configured")
		return
	}
	sla, err := h.slaRepo.GetByAlertID(c.Request.Context(), alertID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "sla not found")
		return
	}
	response.Success(c, sla)
}

func (h *SLAHandler) GetSLAReport(c *gin.Context) {
	var startTime, endTime *time.Time
	if st := c.Query("start_time"); st != "" {
		t, err := time.Parse("2006-01-02", st)
		if err == nil {
			startTime = &t
		}
	}
	if et := c.Query("end_time"); et != "" {
		t, err := time.Parse("2006-01-02", et)
		if err == nil {
			endTime = &t
		}
	}
	_ = startTime
	_ = endTime
	response.Success(c, gin.H{
		"period_start":  nil,
		"period_end":    nil,
		"total_alerts":  0,
		"met_count":     0,
		"breached_count": 0,
		"compliance_rate": 0,
	})
}
