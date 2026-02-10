package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CorrelationHandler struct {
	service *services.AlertCorrelationService
}

func NewCorrelationHandler(service *services.AlertCorrelationService) *CorrelationHandler {
	return &CorrelationHandler{service: service}
}

func (h *CorrelationHandler) AnalyzeCorrelations(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid alert id")
		return
	}

	windowMinutes, _ := strconv.Atoi(c.DefaultQuery("window_minutes", "30"))
	window := time.Duration(windowMinutes) * time.Minute

	result, err := h.service.AnalyzeCorrelations(c.Request.Context(), alertID, window)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *CorrelationHandler) FindPatterns(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	minOccurrences, _ := strconv.Atoi(c.DefaultQuery("min_occurrences", "3"))

	timeRange := time.Duration(hours) * time.Hour
	patterns, err := h.service.FindPatterns(c.Request.Context(), timeRange, minOccurrences)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"data": patterns})
}

func (h *CorrelationHandler) GroupSimilarAlerts(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "1"))
	threshold, _ := strconv.ParseFloat(c.DefaultQuery("threshold", "0.7"), 64)

	timeRange := time.Duration(hours) * time.Hour
	groups, err := h.service.GroupSimilarAlerts(c.Request.Context(), timeRange, threshold)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"data": groups})
}

func (h *CorrelationHandler) GenerateTimeline(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		response.Error(c, http.StatusBadRequest, "fingerprint required")
		return
	}

	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	timeRange := time.Duration(hours) * time.Hour

	events, err := h.service.GenerateTimeline(c.Request.Context(), fingerprint, timeRange)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"data": events})
}

func (h *CorrelationHandler) DetectFlapping(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Query("rule_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule_id")
		return
	}

	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	threshold, _ := strconv.Atoi(c.DefaultQuery("threshold", "5"))

	window := time.Duration(hours) * time.Hour
	flapping, err := h.service.DetectFlapping(c.Request.Context(), ruleID, window, threshold)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"data": flapping})
}

func (h *CorrelationHandler) PredictAlerts(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Param("rule_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule_id")
		return
	}

	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	timeWindow := time.Duration(hours) * time.Hour

	predictions, err := h.service.PredictFutureAlerts(c.Request.Context(), ruleID, timeWindow)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"data": predictions})
}
