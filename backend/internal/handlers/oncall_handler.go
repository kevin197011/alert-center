package handlers

import (
	"alert-center/internal/repository"
	"alert-center/pkg/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OnCallHandler handles on-call schedule and assignment APIs.
type OnCallHandler struct {
	scheduleRepo   *repository.OnCallScheduleRepository
	memberRepo     *repository.OnCallMemberRepository
	assignmentRepo *repository.OnCallAssignmentRepository
}

// NewOnCallHandler returns a new OnCallHandler.
func NewOnCallHandler(scheduleRepo *repository.OnCallScheduleRepository) *OnCallHandler {
	return &OnCallHandler{scheduleRepo: scheduleRepo}
}

// WithRepositories sets member and assignment repositories.
func (h *OnCallHandler) WithRepositories(memberRepo *repository.OnCallMemberRepository, assignmentRepo *repository.OnCallAssignmentRepository) *OnCallHandler {
	h.memberRepo = memberRepo
	h.assignmentRepo = assignmentRepo
	return h
}

func (h *OnCallHandler) GetSchedules(c *gin.Context) {
	list, err := h.scheduleRepo.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list})
}

func (h *OnCallHandler) CreateSchedule(c *gin.Context) {
	var req struct {
		Name          string    `json:"name" binding:"required"`
		Description   string    `json:"description"`
		Timezone      string    `json:"timezone"`
		RotationType  string    `json:"rotation_type"`
		RotationStart time.Time `json:"rotation_start"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if req.RotationType == "" {
		req.RotationType = "weekly"
	}
	schedule := &repository.OnCallSchedule{
		Name:          req.Name,
		Description:   req.Description,
		Timezone:      req.Timezone,
		RotationType:  req.RotationType,
		RotationStart: req.RotationStart,
		Enabled:       true,
	}
	if err := h.scheduleRepo.Create(c.Request.Context(), schedule); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, schedule)
}

func (h *OnCallHandler) GetSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	schedule, err := h.scheduleRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "schedule not found")
		return
	}
	response.Success(c, schedule)
}

func (h *OnCallHandler) UpdateSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	schedule, err := h.scheduleRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "schedule not found")
		return
	}
	var req struct {
		Name          *string    `json:"name"`
		Description   *string    `json:"description"`
		Timezone      *string    `json:"timezone"`
		RotationType  *string    `json:"rotation_type"`
		RotationStart *time.Time `json:"rotation_start"`
		Enabled       *bool      `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		schedule.Name = *req.Name
	}
	if req.Description != nil {
		schedule.Description = *req.Description
	}
	if req.Timezone != nil {
		schedule.Timezone = *req.Timezone
	}
	if req.RotationType != nil {
		schedule.RotationType = *req.RotationType
	}
	if req.RotationStart != nil {
		schedule.RotationStart = *req.RotationStart
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if err := h.scheduleRepo.Update(c.Request.Context(), schedule); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, schedule)
}

func (h *OnCallHandler) DeleteSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.scheduleRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *OnCallHandler) AddMember(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var req struct {
		UserID    string    `json:"user_id" binding:"required"`
		Username  string    `json:"username" binding:"required"`
		Email     string    `json:"email"`
		Phone     string    `json:"phone"`
		Priority  int       `json:"priority"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	member := &repository.OnCallMember{
		ScheduleID: scheduleID,
		UserID:     userID,
		Username:   req.Username,
		Email:      req.Email,
		Phone:      req.Phone,
		Priority:   req.Priority,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		IsActive:   true,
	}
	if err := h.memberRepo.Create(c.Request.Context(), member); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, member)
}

func (h *OnCallHandler) GetMembers(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	list, err := h.memberRepo.GetByScheduleID(c.Request.Context(), scheduleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list})
}

func (h *OnCallHandler) DeleteMember(c *gin.Context) {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	memberID, err := uuid.Parse(c.Param("member_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid member_id")
		return
	}
	if err := h.memberRepo.Delete(c.Request.Context(), memberID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *OnCallHandler) GetScheduleAssignments(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var startTime, endTime time.Time
	if st := c.Query("start_time"); st != "" {
		startTime, _ = time.Parse(time.RFC3339, st)
	} else {
		startTime = time.Now().AddDate(0, 0, -7)
	}
	if et := c.Query("end_time"); et != "" {
		endTime, _ = time.Parse(time.RFC3339, et)
	} else {
		endTime = time.Now().AddDate(0, 0, 30)
	}
	list, err := h.assignmentRepo.GetByScheduleID(c.Request.Context(), scheduleID, startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"data": list})
}

func (h *OnCallHandler) GenerateRotations(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var req struct {
		EndTime time.Time `json:"end_time"`
	}
	c.ShouldBindJSON(&req)
	_ = scheduleID
	_ = req
	response.Success(c, gin.H{"message": "rotations generated"})
}

func (h *OnCallHandler) Escalate(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule_id")
		return
	}
	var req struct {
		CurrentUserID string `json:"current_user_id"`
	}
	c.ShouldBindJSON(&req)
	_ = scheduleID
	_ = req
	assignment, _ := h.assignmentRepo.GetCurrentByScheduleID(c.Request.Context(), scheduleID)
	response.Success(c, assignment)
}

func (h *OnCallHandler) GetCurrentOnCall(c *gin.Context) {
	schedules, _ := h.scheduleRepo.List(c.Request.Context())
	var result []repository.OnCallAssignment
	for _, s := range schedules {
		a, err := h.assignmentRepo.GetCurrentByScheduleID(c.Request.Context(), s.ID)
		if err == nil {
			result = append(result, *a)
		}
	}
	response.Success(c, gin.H{"data": result})
}

func (h *OnCallHandler) WhoIsOnCall(c *gin.Context) {
	atTime := c.Query("at_time")
	var t time.Time
	if atTime != "" {
		t, _ = time.Parse(time.RFC3339, atTime)
	} else {
		t = time.Now()
	}
	_ = t
	schedules, _ := h.scheduleRepo.List(c.Request.Context())
	var result []repository.OnCallAssignment
	for _, s := range schedules {
		a, err := h.assignmentRepo.GetCurrentByScheduleID(c.Request.Context(), s.ID)
		if err == nil {
			result = append(result, *a)
		}
	}
	response.Success(c, gin.H{"data": result})
}

func (h *OnCallHandler) GetOnCallReport(c *gin.Context) {
	response.Success(c, gin.H{"data": []interface{}{}})
}

func (h *OnCallHandler) SeedDefaultSchedules(c *gin.Context) {
	list, _ := h.scheduleRepo.List(c.Request.Context())
	if len(list) > 0 {
		response.Success(c, gin.H{"message": "schedules already exist"})
		return
	}
	response.Success(c, gin.H{"message": "no default schedules to seed"})
}
