package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AlertChannelBindingHandler struct {
	service *services.AlertChannelBindingService
}

func NewAlertChannelBindingHandler(service *services.AlertChannelBindingService) *AlertChannelBindingHandler {
	return &AlertChannelBindingHandler{service: service}
}

func (h *AlertChannelBindingHandler) BindChannels(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule id")
		return
	}

	var req struct {
		ChannelIDs []uuid.UUID `json:"channel_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.BindChannels(c.Request.Context(), ruleID, req.ChannelIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "bind success"})
}

func (h *AlertChannelBindingHandler) GetBindings(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule id")
		return
	}

	channels, err := h.service.GetByRuleID(c.Request.Context(), ruleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, channels)
}
