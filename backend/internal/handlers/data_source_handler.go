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

type DataSourceHandler struct {
	service *services.DataSourceService
}

func NewDataSourceHandler(service *services.DataSourceService) *DataSourceHandler {
	return &DataSourceHandler{service: service}
}

func (h *DataSourceHandler) Create(c *gin.Context) {
	var req services.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ds, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, ds)
}

func (h *DataSourceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	list, total, err := h.service.List(c.Request.Context(), page, pageSize, c.Query("type"), -1)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":  list,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *DataSourceHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	ds, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "data source not found")
		return
	}
	response.Success(c, ds)
}

func (h *DataSourceHandler) HealthCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.HealthCheck(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "health check completed"})
}

func (h *DataSourceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req services.UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ds, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, ds)
}

func (h *DataSourceHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

type AlertStatisticsHandler struct {
	service *services.AlertStatisticsService
}

func NewAlertStatisticsHandler(service *services.AlertStatisticsService) *AlertStatisticsHandler {
	return &AlertStatisticsHandler{service: service}
}

func (h *AlertStatisticsHandler) Statistics(c *gin.Context) {
	startTime, endTime := parseTimeRange(c)
	var groupID *string
	if g := c.Query("group_id"); g != "" {
		groupID = &g
	}
	stats, err := h.service.GetStatistics(c.Request.Context(), startTime, endTime, groupID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

func (h *AlertStatisticsHandler) Dashboard(c *gin.Context) {
	summary, err := h.service.GetDashboardSummary(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, summary)
}

func parseTimeRange(c *gin.Context) (startTime, endTime *time.Time) {
	const layout = "2006-01-02"
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(layout, st); err == nil {
			startTime = &t
		}
	}
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(layout, et); err == nil {
			endTime = &t
		}
	}
	return
}
