package handlers

import (
	"alert-center/internal/repository"
	"alert-center/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EscalationHistoryHandler handles escalation history (user_escalations list) APIs.
type EscalationHistoryHandler struct {
	db *repository.Database
}

// NewEscalationHistoryHandler returns a new EscalationHistoryHandler.
func NewEscalationHistoryHandler(db *repository.Database) *EscalationHistoryHandler {
	return &EscalationHistoryHandler{db: db}
}

func (h *EscalationHistoryHandler) GetHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	rows, err := h.db.Pool.Query(c.Request.Context(), `
		SELECT id, alert_id, from_user_id, from_username, to_user_id, to_username, reason, status, created_at, resolved_at
		FROM user_escalations ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, alertID, fromUserID, toUserID uuid.UUID
		var fromUsername, toUsername, reason, status string
		var createdAt interface{}
		var resolvedAt interface{}
		if err := rows.Scan(&id, &alertID, &fromUserID, &fromUsername, &toUserID, &toUsername, &reason, &status, &createdAt, &resolvedAt); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{
			"id": id, "alert_id": alertID, "from_user_id": fromUserID, "from_username": fromUsername,
			"to_user_id": toUserID, "to_username": toUsername, "reason": reason, "status": status,
			"created_at": createdAt, "resolved_at": resolvedAt,
		})
	}
	var total int
	_ = h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM user_escalations`).Scan(&total)
	response.Success(c, gin.H{"data": list, "total": total, "page": page, "size": pageSize})
}

func (h *EscalationHistoryHandler) GetStats(c *gin.Context) {
	var pending, accepted, rejected, resolved int
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM user_escalations WHERE status = 'pending'`).Scan(&pending)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM user_escalations WHERE status = 'accepted'`).Scan(&accepted)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM user_escalations WHERE status = 'rejected'`).Scan(&rejected)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM user_escalations WHERE status = 'resolved'`).Scan(&resolved)
	response.Success(c, gin.H{
		"pending": pending, "accepted": accepted, "rejected": rejected, "resolved": resolved,
		"total": pending + accepted + rejected + resolved,
	})
}

func (h *EscalationHistoryHandler) GetByAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("alert_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid alert_id")
		return
	}
	rows, err := h.db.Pool.Query(c.Request.Context(), `
		SELECT id, alert_id, from_user_id, from_username, to_user_id, to_username, reason, status, created_at, resolved_at
		FROM user_escalations WHERE alert_id = $1 ORDER BY created_at DESC
	`, alertID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, aid, fromUserID, toUserID uuid.UUID
		var fromUsername, toUsername, reason, status string
		var createdAt, resolvedAt interface{}
		if err := rows.Scan(&id, &aid, &fromUserID, &fromUsername, &toUserID, &toUsername, &reason, &status, &createdAt, &resolvedAt); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{
			"id": id, "alert_id": aid, "from_user_id": fromUserID, "from_username": fromUsername,
			"to_user_id": toUserID, "to_username": toUsername, "reason": reason, "status": status,
			"created_at": createdAt, "resolved_at": resolvedAt,
		})
	}
	response.Success(c, gin.H{"data": list})
}
