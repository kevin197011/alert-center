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

type UserManagementHandler struct {
	service *services.UserManagementService
}

func NewUserManagementHandler(service *services.UserManagementService) *UserManagementHandler {
	return &UserManagementHandler{service: service}
}

func (h *UserManagementHandler) Create(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *UserManagementHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "user not found")
		return
	}

	response.Success(c, user)
}

func (h *UserManagementHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, total, err := h.service.List(c.Request.Context(), page, pageSize, c.Query("role"), c.Query("status"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":  users,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *UserManagementHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *UserManagementHandler) Delete(c *gin.Context) {
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

func (h *UserManagementHandler) ChangePassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), id, req.OldPassword, req.NewPassword); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "password changed successfully"})
}

type AuditLogHandler struct {
	service *services.AuditLogService
}

func NewAuditLogHandler(service *services.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{service: service}
}

func (h *AuditLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var startTime, endTime *time.Time
	if st := c.Query("start_time"); st != "" {
		t, _ := time.Parse("2006-01-02", st)
		startTime = &t
	}
	if et := c.Query("end_time"); et != "" {
		t, _ := time.Parse("2006-01-02", et)
		endTime = &t
	}

	var userID *uuid.UUID
	if uid := c.Query("user_id"); uid != "" {
		id, _ := uuid.Parse(uid)
		userID = &id
	}

	req := &services.ListAuditLogRequest{
		UserID:    userID,
		Action:    c.Query("action"),
		Resource:  c.Query("resource"),
		StartTime: startTime,
		EndTime:   endTime,
	}

	logs, total, err := h.service.List(c.Request.Context(), page, pageSize, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *AuditLogHandler) Export(c *gin.Context) {
	var startTime, endTime *time.Time
	if st := c.Query("start_time"); st != "" {
		t, _ := time.Parse("2006-01-02", st)
		startTime = &t
	}
	if et := c.Query("end_time"); et != "" {
		t, _ := time.Parse("2006-01-02", et)
		endTime = &t
	}

	var userID *uuid.UUID
	if uid := c.Query("user_id"); uid != "" {
		id, _ := uuid.Parse(uid)
		userID = &id
	}

	req := &services.ListAuditLogRequest{
		UserID:    userID,
		Action:    c.Query("action"),
		Resource:  c.Query("resource"),
		StartTime: startTime,
		EndTime:   endTime,
	}

	logs, err := h.service.Export(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.json")
	c.JSON(http.StatusOK, logs)
}
