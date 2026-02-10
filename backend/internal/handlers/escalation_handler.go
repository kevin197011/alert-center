package handlers

import (
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EscalationHandler handles user escalation (handoff) APIs.
type EscalationHandler struct {
	service *services.AlertEscalationService
}

// NewEscalationHandler returns a new EscalationHandler.
func NewEscalationHandler(svc *services.AlertEscalationService) *EscalationHandler {
	return &EscalationHandler{service: svc}
}

func (h *EscalationHandler) CreateEscalation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	var req services.CreateEscalationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	esc, err := h.service.CreateEscalation(c.Request.Context(), userID.(uuid.UUID), username.(string), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, esc)
}

func (h *EscalationHandler) GetAlertEscalations(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("alert_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid alert_id")
		return
	}
	list, err := h.service.GetAlertEscalations(c.Request.Context(), alertID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list})
}

func (h *EscalationHandler) GetMyPendingEscalations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	list, err := h.service.GetPendingEscalations(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list})
}

func (h *EscalationHandler) AcceptEscalation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.AcceptEscalation(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "accepted"})
}

func (h *EscalationHandler) RejectEscalation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.RejectEscalation(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "rejected"})
}

func (h *EscalationHandler) ResolveEscalation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.ResolveEscalation(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "resolved"})
}
