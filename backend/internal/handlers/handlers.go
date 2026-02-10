package handlers

import (
	"alert-center/internal/models"
	"alert-center/internal/repository"
	"alert-center/internal/services"
	"alert-center/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	response.Success(c, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.service.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		response.Error(c, http.StatusNotFound, "user not found")
		return
	}
	response.Success(c, user)
}

type AlertRuleHandler struct {
	service        *services.AlertRuleService
	bindingService *services.AlertChannelBindingService
}

func NewAlertRuleHandler(service *services.AlertRuleService, bindingService *services.AlertChannelBindingService) *AlertRuleHandler {
	return &AlertRuleHandler{service: service, bindingService: bindingService}
}

func (h *AlertRuleHandler) Create(c *gin.Context) {
	var req services.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, rule)
}

// TestExpressionRequest is the body for testing a PromQL expression against a data source.
type TestExpressionRequest struct {
	Expression      string `json:"expression" binding:"required"`
	DataSourceType  string `json:"data_source_type"`
	DataSourceURL   string `json:"data_source_url" binding:"required"`
}

func (h *AlertRuleHandler) TestExpression(c *gin.Context) {
	var req TestExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.DataSourceURL == "" {
		response.Error(c, http.StatusBadRequest, "data_source_url is required")
		return
	}

	ctx := c.Request.Context()
	var results []models.QueryResult
	var err error
	switch req.DataSourceType {
	case "victoria-metrics":
		vm := services.NewVictoriaMetricsClient(req.DataSourceURL)
		results, err = vm.Query(ctx, req.Expression, "")
	default:
		prom := services.NewPrometheusClient(req.DataSourceURL)
		results, err = prom.Query(ctx, req.Expression, "")
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, gin.H{
		"result_type": "vector",
		"count":       len(results),
		"data":        results,
	})
}

func (h *AlertRuleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	rule, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
		return
	}

	response.Success(c, rule)
}

func (h *AlertRuleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	req := &services.ListAlertRuleRequest{
		Page:     page,
		PageSize: pageSize,
		GroupID:  c.Query("group_id"),
		Severity: c.Query("severity"),
		Status:   c.Query("status"),
	}

	rules, total, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	ruleIDs := make([]uuid.UUID, 0, len(rules))
	for _, r := range rules {
		ruleIDs = append(ruleIDs, r.ID)
	}
	channelsByRule, _ := h.bindingService.GetChannelsByRuleIDs(c.Request.Context(), ruleIDs)
	type ruleWithChannels struct {
		models.AlertRule
		BoundChannels []models.AlertChannel `json:"bound_channels"`
	}
	list := make([]ruleWithChannels, 0, len(rules))
	for _, r := range rules {
		channels := channelsByRule[r.ID]
		if channels == nil {
			channels = []models.AlertChannel{}
		}
		list = append(list, ruleWithChannels{AlertRule: r, BoundChannels: channels})
	}

	response.Success(c, gin.H{
		"data":  list,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *AlertRuleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req services.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, rule)
}

func (h *AlertRuleHandler) Delete(c *gin.Context) {
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

func (h *AlertRuleHandler) Export(c *gin.Context) {
	req := &services.StatisticsRequest{
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=alert_statistics.json")
	c.JSON(http.StatusOK, stats)
}

func (h *AlertRuleHandler) GetBindings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	channels, err := h.bindingService.GetByRuleID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, channels)
}

type AlertChannelHandler struct {
	service *services.AlertChannelService
}

func NewAlertChannelHandler(service *services.AlertChannelService) *AlertChannelHandler {
	return &AlertChannelHandler{service: service}
}

func (h *AlertChannelHandler) Create(c *gin.Context) {
	var req services.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	channel, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, channel)
}

func (h *AlertChannelHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	req := &services.ListChannelRequest{
		Page:     page,
		PageSize: pageSize,
		Type:     c.Query("type"),
		Status:   c.Query("status"),
	}

	channels, total, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":  channels,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *AlertChannelHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	channel, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "channel not found")
		return
	}

	response.Success(c, channel)
}

func (h *AlertChannelHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req services.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	channel, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, channel)
}

func (h *AlertChannelHandler) Delete(c *gin.Context) {
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

func (h *AlertChannelHandler) Test(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.SendTest(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "test sent"})
}

// TestWithConfigRequest is the body for testing a channel with type and config (e.g. before save).
type TestWithConfigRequest struct {
	Type   string                 `json:"type" binding:"required"`
	Config map[string]interface{} `json:"config" binding:"required"`
}

func (h *AlertChannelHandler) TestWithConfig(c *gin.Context) {
	var req TestWithConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.SendTestWithConfig(c.Request.Context(), req.Type, req.Config); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "test sent"})
}

type BusinessGroupHandler struct {
	repo *repository.BusinessGroupRepository
}

func NewBusinessGroupHandler(repo *repository.BusinessGroupRepository) *BusinessGroupHandler {
	return &BusinessGroupHandler{repo: repo}
}

func (h *BusinessGroupHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))

	groups, total, err := h.repo.List(c.Request.Context(), page, pageSize, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":  groups,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

type AlertHistoryHandler struct {
	repo *repository.AlertHistoryRepository
}

func NewAlertHistoryHandler(repo *repository.AlertHistoryRepository) *AlertHistoryHandler {
	return &AlertHistoryHandler{repo: repo}
}

func (h *AlertHistoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var ruleID *uuid.UUID
	if ruleIDStr := c.Query("rule_id"); ruleIDStr != "" {
		id, err := uuid.Parse(ruleIDStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid rule_id")
			return
		}
		ruleID = &id
	}

	histories, total, err := h.repo.List(c.Request.Context(), page, pageSize, ruleID, c.Query("status"), nil, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":  histories,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}
