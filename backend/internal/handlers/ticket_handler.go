package handlers

import (
	"alert-center/internal/repository"
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TicketHandler handles ticket APIs.
type TicketHandler struct {
	db          *repository.Database
	broadcaster services.Broadcaster
}

// NewTicketHandler returns a new TicketHandler.
func NewTicketHandler(db *repository.Database, broadcaster services.Broadcaster) *TicketHandler {
	return &TicketHandler{db: db, broadcaster: broadcaster}
}

func (h *TicketHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	q := `SELECT id, title, description, alert_id, rule_id, priority, status, assignee_id, assignee_name, creator_id, creator_name, created_at, updated_at, resolved_at, closed_at FROM tickets WHERE 1=1`
	args := []interface{}{}
	n := 1
	if status != "" {
		q += ` AND status = $` + strconv.Itoa(n)
		args = append(args, status)
		n++
	}
	q += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1)
	args = append(args, pageSize, offset)
	rows, err := h.db.Pool.Query(c.Request.Context(), q, args...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, creatorID uuid.UUID
		var title, description, priority, status, creatorName string
		var alertID, ruleID, assigneeID *uuid.UUID
		var assigneeName *string
		var createdAt, updatedAt time.Time
		var resolvedAt, closedAt *time.Time
		if err := rows.Scan(&id, &title, &description, &alertID, &ruleID, &priority, &status, &assigneeID, &assigneeName, &creatorID, &creatorName, &createdAt, &updatedAt, &resolvedAt, &closedAt); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{
			"id": id, "title": title, "description": description, "alert_id": alertID, "rule_id": ruleID,
			"priority": priority, "status": status, "assignee_id": assigneeID, "assignee_name": assigneeName,
			"creator_id": creatorID, "creator_name": creatorName, "created_at": createdAt, "updated_at": updatedAt,
			"resolved_at": resolvedAt, "closed_at": closedAt,
		})
	}
	var total int
	countQ := `SELECT COUNT(*) FROM tickets WHERE 1=1`
	if status != "" {
		h.db.Pool.QueryRow(c.Request.Context(), countQ+` AND status = $1`, status).Scan(&total)
	} else {
		h.db.Pool.QueryRow(c.Request.Context(), countQ).Scan(&total)
	}
	response.Success(c, gin.H{"data": list, "total": total, "page": page, "size": pageSize})
}

func (h *TicketHandler) Create(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	var req struct {
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		AlertID     *string `json:"alert_id"`
		RuleID      *string `json:"rule_id"`
		Priority    string  `json:"priority"`
		AssigneeName string `json:"assignee_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	id := uuid.New()
	now := time.Now()
	var alertID, ruleID *uuid.UUID
	if req.AlertID != nil && *req.AlertID != "" {
		a, _ := uuid.Parse(*req.AlertID)
		alertID = &a
	}
	if req.RuleID != nil && *req.RuleID != "" {
		r, _ := uuid.Parse(*req.RuleID)
		ruleID = &r
	}
	_, err := h.db.Pool.Exec(c.Request.Context(), `
		INSERT INTO tickets (id, title, description, alert_id, rule_id, priority, status, assignee_name, creator_id, creator_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'open', $7, $8, $9, $10, $10)
	`, id, req.Title, req.Description, alertID, ruleID, req.Priority, req.AssigneeName, userID.(uuid.UUID), username.(string), now)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "title": req.Title, "status": "open", "created_at": now})
	if h.broadcaster != nil {
		h.broadcaster.SendTicketNotification(&services.TicketNotification{
			TicketID:  id.String(),
			Title:     req.Title,
			Status:    "open",
			Action:    "created",
			Timestamp: now,
		})
	}
}

func (h *TicketHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var title, description, priority, status, creatorName string
	var alertID, ruleID, assigneeID *uuid.UUID
	var assigneeName *string
	var creatorID uuid.UUID
	var createdAt, updatedAt time.Time
	var resolvedAt, closedAt *time.Time
	var rowID uuid.UUID
	err = h.db.Pool.QueryRow(c.Request.Context(), `
		SELECT id, title, description, alert_id, rule_id, priority, status, assignee_id, assignee_name, creator_id, creator_name, created_at, updated_at, resolved_at, closed_at
		FROM tickets WHERE id = $1
	`, id).Scan(&rowID, &title, &description, &alertID, &ruleID, &priority, &status, &assigneeID, &assigneeName, &creatorID, &creatorName, &createdAt, &updatedAt, &resolvedAt, &closedAt)
	if err != nil {
		response.Error(c, http.StatusNotFound, "ticket not found")
		return
	}
	response.Success(c, gin.H{
		"id": rowID, "title": title, "description": description, "alert_id": alertID, "rule_id": ruleID,
		"priority": priority, "status": status, "assignee_id": assigneeID, "assignee_name": assigneeName,
		"creator_id": creatorID, "creator_name": creatorName, "created_at": createdAt, "updated_at": updatedAt,
		"resolved_at": resolvedAt, "closed_at": closedAt,
	})
}

func (h *TicketHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Priority    *string `json:"priority"`
		Status      *string `json:"status"`
		AssigneeID  *string `json:"assignee_id"`
		AssigneeName *string `json:"assignee_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	_, err = h.db.Pool.Exec(c.Request.Context(), `UPDATE tickets SET updated_at = $1 WHERE id = $2`, time.Now(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "message": "updated"})
	if h.broadcaster != nil {
		h.broadcaster.SendTicketNotification(&services.TicketNotification{
			TicketID:  id.String(),
			Title:     "",
			Status:    "updated",
			Action:    "updated",
			Timestamp: time.Now(),
		})
	}
}

func (h *TicketHandler) Resolve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	now := time.Now()
	_, err = h.db.Pool.Exec(c.Request.Context(), `UPDATE tickets SET status = 'resolved', resolved_at = $1, updated_at = $1 WHERE id = $2`, now, id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "resolved"})
	if h.broadcaster != nil {
		h.broadcaster.SendTicketNotification(&services.TicketNotification{
			TicketID:  id.String(),
			Title:     "",
			Status:    "resolved",
			Action:    "resolved",
			Timestamp: now,
		})
	}
}

func (h *TicketHandler) Close(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	now := time.Now()
	_, err = h.db.Pool.Exec(c.Request.Context(), `UPDATE tickets SET status = 'closed', closed_at = $1, updated_at = $1 WHERE id = $2`, now, id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "closed"})
	if h.broadcaster != nil {
		h.broadcaster.SendTicketNotification(&services.TicketNotification{
			TicketID:  id.String(),
			Title:     "",
			Status:    "closed",
			Action:    "closed",
			Timestamp: now,
		})
	}
}

func (h *TicketHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	_, err = h.db.Pool.Exec(c.Request.Context(), `DELETE FROM tickets WHERE id = $1`, id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, nil)
	if h.broadcaster != nil {
		h.broadcaster.SendTicketNotification(&services.TicketNotification{
			TicketID:  id.String(),
			Title:     "",
			Status:    "deleted",
			Action:    "deleted",
			Timestamp: time.Now(),
		})
	}
}

func (h *TicketHandler) Stats(c *gin.Context) {
	var open, inProgress, resolved, closed, total int
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM tickets WHERE status = 'open'`).Scan(&open)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM tickets WHERE status = 'in_progress'`).Scan(&inProgress)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM tickets WHERE status = 'resolved'`).Scan(&resolved)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM tickets WHERE status = 'closed'`).Scan(&closed)
	h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM tickets`).Scan(&total)
	response.Success(c, gin.H{"data": gin.H{"open": open, "in_progress": inProgress, "resolved": resolved, "closed": closed, "total": total}})
}
