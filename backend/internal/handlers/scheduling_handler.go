package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SchedulingHandler handles on-call schedule generation and coverage APIs.
type SchedulingHandler struct {
	service *services.SchedulingService
}

// NewSchedulingHandler returns a new SchedulingHandler.
func NewSchedulingHandler(service *services.SchedulingService) *SchedulingHandler {
	return &SchedulingHandler{service: service}
}

func (h *SchedulingHandler) GenerateSchedule(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var req struct {
		StartTime     string `json:"start_time" binding:"required"`
		EndTime       string `json:"end_time" binding:"required"`
		ShiftDuration int    `json:"shift_duration"`
		Timezone      string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		startTime, _ = time.Parse("2006-01-02", req.StartTime)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		endTime, _ = time.Parse("2006-01-02", req.EndTime)
	}
	if req.ShiftDuration <= 0 {
		req.ShiftDuration = 24
	}
	shifts, total, err := h.service.GenerateSchedule(c.Request.Context(), scheduleID, startTime, endTime, req.ShiftDuration, req.Timezone)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"shifts": shifts, "total": total})
}

func (h *SchedulingHandler) GetScheduleCoverage(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var startTime, endTime *time.Time
	if st := c.Query("start_time"); st != "" {
		t, _ := time.Parse(time.RFC3339, st)
		startTime = &t
	}
	if et := c.Query("end_time"); et != "" {
		t, _ := time.Parse(time.RFC3339, et)
		endTime = &t
	}
	gaps, total, err := h.service.GetScheduleCoverage(c.Request.Context(), scheduleID, startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"gaps": gaps, "total_gaps": total})
}

func (h *SchedulingHandler) SuggestRotation(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	suggestions, err := h.service.SuggestRotation(c.Request.Context(), scheduleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"suggestions": suggestions})
}

func (h *SchedulingHandler) ValidateSchedule(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var startTime, endTime *time.Time
	if st := c.Query("start_time"); st != "" {
		t, _ := time.Parse(time.RFC3339, st)
		startTime = &t
	}
	if et := c.Query("end_time"); et != "" {
		t, _ := time.Parse(time.RFC3339, et)
		endTime = &t
	}
	validation, err := h.service.ValidateSchedule(c.Request.Context(), scheduleID, startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, validation)
}
