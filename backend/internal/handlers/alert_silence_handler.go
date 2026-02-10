package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AlertSilenceHandler struct {
	service *services.AlertSilenceService
}

func NewAlertSilenceHandler(service *services.AlertSilenceService) *AlertSilenceHandler {
	return &AlertSilenceHandler{service: service}
}

func (h *AlertSilenceHandler) Create(c *gin.Context) {
	var req services.CreateSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("user_id")

	silence, err := h.service.Create(c.Request.Context(), &req, userID.(uuid.UUID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, silence)
}

func (h *AlertSilenceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))

	list, total, err := h.service.List(c.Request.Context(), page, pageSize, status)
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

func (h *AlertSilenceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req services.UpdateSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	silence, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, silence)
}

func (h *AlertSilenceHandler) Delete(c *gin.Context) {
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

func (h *AlertSilenceHandler) Check(c *gin.Context) {
	var req struct {
		Labels map[string]string `json:"labels" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	silenced, err := h.service.IsSilenced(c.Request.Context(), req.Labels)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"silenced": silenced})
}
