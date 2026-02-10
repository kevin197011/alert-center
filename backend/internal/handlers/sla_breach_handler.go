package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SLABreachHandler handles SLA breach APIs.
type SLABreachHandler struct {
	service *services.SLABreachService
}

// NewSLABreachHandler returns a new SLABreachHandler.
func NewSLABreachHandler(service *services.SLABreachService) *SLABreachHandler {
	return &SLABreachHandler{service: service}
}

func (h *SLABreachHandler) GetBreaches(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	list, total, err := h.service.GetBreaches(c.Request.Context(), page, pageSize, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list, "total": total})
}

func (h *SLABreachHandler) GetBreachStats(c *gin.Context) {
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
	stats, err := h.service.GetBreachStats(c.Request.Context(), startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

func (h *SLABreachHandler) TriggerCheck(c *gin.Context) {
	count, err := h.service.TriggerCheck(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"breaches_found": count})
}

func (h *SLABreachHandler) TriggerNotifications(c *gin.Context) {
	count, err := h.service.TriggerNotifications(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"notifications": count})
}
