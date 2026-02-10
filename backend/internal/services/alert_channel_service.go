package services

import (
	"alert-center/internal/models"
	"alert-center/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AlertChannelService struct {
	repo *repository.AlertChannelRepository
}

func NewAlertChannelService(repo *repository.AlertChannelRepository) *AlertChannelService {
	return &AlertChannelService{repo: repo}
}

func (s *AlertChannelService) Create(ctx context.Context, req *CreateChannelRequest) (*models.AlertChannel, error) {
	config, _ := json.Marshal(req.Config)

	channel := &models.AlertChannel{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Config:      string(config),
		GroupID:     req.GroupID,
		Status:      1,
	}

	if err := s.repo.Create(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

func (s *AlertChannelService) List(ctx context.Context, req *ListChannelRequest) ([]models.AlertChannel, int, error) {
	var status int
	if req.Status != "" {
		status = 1 // enabled
	} else if req.Status == "disabled" {
		status = 0
	} else {
		status = -1 // all
	}

	return s.repo.List(ctx, req.Page, req.PageSize, req.Type, status)
}

func (s *AlertChannelService) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertChannel, error) {
	channels, _, err := s.repo.List(ctx, 1, 1, "", 1)
	if err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if ch.ID == id {
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("channel not found")
}

func (s *AlertChannelService) Update(ctx context.Context, id uuid.UUID, req *UpdateChannelRequest) (*models.AlertChannel, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil || channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Type != nil {
		channel.Type = *req.Type
	}
	if req.Description != nil {
		channel.Description = *req.Description
	}
	if req.Config != nil {
		config, _ := json.Marshal(req.Config)
		channel.Config = string(config)
	}
	if req.GroupID != nil {
		channel.GroupID = req.GroupID
	}

	if err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}
	return channel, nil
}

func (s *AlertChannelService) Delete(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement soft delete
	return nil
}

// SendTestWithConfig sends a test notification using the given type and config (for testing before save).
func (s *AlertChannelService) SendTestWithConfig(ctx context.Context, channelType string, config map[string]interface{}) error {
	if config == nil {
		config = make(map[string]interface{})
	}
	testPayload := &AlertPayload{
		AlertNo:     "AL-TEST",
		RuleID:      uuid.Nil,
		RuleName:    "【测试】告警渠道连通性",
		Severity:    "info",
		Status:      "firing",
		Description: "这是一条测试消息，用于验证渠道配置是否正确。",
		Labels:      "{}",
		StartedAt:   time.Now(),
	}
	switch channelType {
	case "lark":
		return s.sendLark(ctx, config, testPayload)
	case "telegram":
		return s.sendTelegram(ctx, config, testPayload)
	case "webhook":
		return s.sendWebhook(ctx, config, testPayload)
	default:
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// SendTest sends a test notification to the channel for connectivity verification.
func (s *AlertChannelService) SendTest(ctx context.Context, channelID uuid.UUID) error {
	channels, _, err := s.repo.List(ctx, 1, 100, "", -1)
	if err != nil {
		return err
	}
	var channel *models.AlertChannel
	for i := range channels {
		if channels[i].ID == channelID {
			channel = &channels[i]
			break
		}
	}
	if channel == nil {
		return fmt.Errorf("channel not found")
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("invalid channel config")
	}
	return s.SendTestWithConfig(ctx, channel.Type, config)
}

func (s *AlertChannelService) Send(ctx context.Context, channelID uuid.UUID, alert *AlertPayload) error {
	channels, _, err := s.repo.List(ctx, 1, 1, "", 1)
	if err != nil {
		return err
	}

	var channel models.AlertChannel
	for _, ch := range channels {
		if ch.ID == channelID {
			channel = ch
			break
		}
	}

	var config map[string]interface{}
	json.Unmarshal([]byte(channel.Config), &config)

	switch channel.Type {
	case "lark":
		return s.sendLark(ctx, config, alert)
	case "telegram":
		return s.sendTelegram(ctx, config, alert)
	case "webhook":
		return s.sendWebhook(ctx, config, alert)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

func larkCardHeaderTemplate(severity string) string {
	switch severity {
	case "critical":
		return "red"
	case "warning":
		return "orange"
	default:
		return "blue"
	}
}

func buildLarkCardPayload(alert *AlertPayload) map[string]interface{} {
	headerTitle := "告警通知"
	if alert.Status == "resolved" {
		headerTitle = "告警恢复"
	}
	// When rule has a template, use rendered content as the card body.
	if alert.RenderedContent != "" {
		return map[string]interface{}{
			"msg_type": "interactive",
			"card": map[string]interface{}{
				"config": map[string]interface{}{
					"wide_screen_mode": true,
				},
				"header": map[string]interface{}{
					"template": larkCardHeaderTemplate(alert.Severity),
					"title": map[string]interface{}{
						"content": headerTitle,
						"tag":    "plain_text",
					},
				},
				"elements": []map[string]interface{}{
					{
						"tag": "div",
						"text": map[string]interface{}{
							"content": alert.RenderedContent,
							"tag":    "lark_md",
						},
					},
				},
			},
		}
	}
	timeStr := alert.StartedAt.Format("2006-01-02 15:04:05")
	desc := alert.Description
	if desc == "" {
		desc = "-"
	}
	alertNoStr := alert.AlertNo
	if alertNoStr == "" {
		alertNoStr = "-"
	}
	elements := []map[string]interface{}{
		{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**告警编号**\n" + alertNoStr,
				"tag":    "lark_md",
			},
		},
		{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**规则名称**\n" + alert.RuleName,
				"tag":    "lark_md",
			},
		},
		{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**严重级别**\n" + alert.Severity,
				"tag":    "lark_md",
			},
		},
		{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**状态**\n" + alert.Status,
				"tag":    "lark_md",
			},
		},
		{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**开始时间**\n" + timeStr,
				"tag":    "lark_md",
			},
		},
		{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**描述**\n" + desc,
				"tag":    "lark_md",
			},
		},
	}
	if alert.Status == "resolved" && alert.EndedAt != nil {
		headerTitle = "告警恢复"
		endStr := alert.EndedAt.Format("2006-01-02 15:04:05")
		dur := alert.EndedAt.Sub(alert.StartedAt).Round(time.Second)
		elements = append(elements,
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**恢复时间**\n" + endStr,
					"tag":    "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**持续时长**\n" + dur.String(),
					"tag":    "lark_md",
				},
			},
		)
	}
	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"header": map[string]interface{}{
				"template": larkCardHeaderTemplate(alert.Severity),
				"title": map[string]interface{}{
					"content": headerTitle,
					"tag":    "plain_text",
				},
			},
			"elements": elements,
		},
	}
}

func (s *AlertChannelService) sendLark(ctx context.Context, config map[string]interface{}, alert *AlertPayload) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("lark webhook_url not configured")
	}

	payload := buildLarkCardPayload(alert)
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lark webhook failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	// Lark returns 200 with body {"code":0,"msg":"success"} on success, or {"code":19002,"msg":"params error"} on failure
	var larkResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &larkResp); err == nil && larkResp.Code != 0 {
		return fmt.Errorf("lark webhook rejected: %s (code %d)", larkResp.Msg, larkResp.Code)
	}
	return nil
}

func (s *AlertChannelService) sendTelegram(ctx context.Context, config map[string]interface{}, alert *AlertPayload) error {
	botToken, ok := config["bot_token"].(string)
	if !ok {
		return fmt.Errorf("telegram bot_token not configured")
	}
	chatID, ok := config["chat_id"].(string)
	if !ok {
		return fmt.Errorf("telegram chat_id not configured")
	}

	var text string
	if alert.RenderedContent != "" {
		if alert.Status == "resolved" {
			text = "✅ *告警恢复*\n\n" + alert.RenderedContent
		} else {
			text = "🚨 *告警通知*\n\n" + alert.RenderedContent
		}
	} else {
		alertNoStr := alert.AlertNo
		if alertNoStr == "" {
			alertNoStr = "-"
		}
		if alert.Status == "resolved" && alert.EndedAt != nil {
			dur := alert.EndedAt.Sub(alert.StartedAt).Round(time.Second)
			text = fmt.Sprintf("✅ *告警恢复*\n\n*告警编号*: %s\n*规则名称*: %s\n*严重级别*: %s\n*状态*: %s\n*开始时间*: %s\n*恢复时间*: %s\n*持续时长*: %s\n\n*描述*: %s",
				alertNoStr, alert.RuleName, alert.Severity, alert.Status,
				alert.StartedAt.Format("2006-01-02 15:04:05"),
				alert.EndedAt.Format("2006-01-02 15:04:05"), dur.String(), alert.Description)
		} else {
			text = fmt.Sprintf("🚨 *告警通知*\n\n*告警编号*: %s\n*规则名称*: %s\n*严重级别*: %s\n*状态*: %s\n*开始时间*: %s\n\n*描述*: %s",
				alertNoStr, alert.RuleName, alert.Severity, alert.Status,
				alert.StartedAt.Format("2006-01-02 15:04:05"), alert.Description)
		}
	}

	base := telegramAPIBase()
	if v, ok := config["api_base"].(string); ok && v != "" {
		base = strings.TrimRight(v, "/")
	}
	url := fmt.Sprintf("%s/bot%s/sendMessage", base, botToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram send failed: %s", string(respBody))
	}

	return nil
}


func (s *AlertChannelService) sendWebhook(ctx context.Context, config map[string]interface{}, alert *AlertPayload) error {
	webhookURL, ok := config["url"].(string)
	if !ok {
		return fmt.Errorf("webhook url not configured")
	}

	var body []byte
	if isLarkWebhookURL(webhookURL) {
		payload := buildLarkCardPayload(alert)
		body, _ = json.Marshal(payload)
	} else {
		body, _ = json.Marshal(alert)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	// Lark webhook returns 200 with body {"code":0} on success, or {"code":19002,"msg":"..."} on failure
	if isLarkWebhookURL(webhookURL) {
		var larkResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.Unmarshal(respBody, &larkResp); err == nil && larkResp.Code != 0 {
			return fmt.Errorf("lark webhook rejected: %s (code %d)", larkResp.Msg, larkResp.Code)
		}
	}
	return nil
}

// isLarkWebhookURL returns true if the URL is a Lark/Feishu robot webhook (which requires msg_type in body).
func isLarkWebhookURL(url string) bool {
	return strings.Contains(url, "larksuite.com") && strings.Contains(url, "open-apis/bot/v2/hook")
}

type CreateChannelRequest struct {
	Name        string             `json:"name" binding:"required"`
	Type        string             `json:"type" binding:"required"`
	Description string             `json:"description"`
	Config      map[string]interface{} `json:"config" binding:"required"`
	GroupID     *uuid.UUID         `json:"group_id"`
}

type ListChannelRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Type     string `form:"type"`
	Status   string `form:"status"`
}

type UpdateChannelRequest struct {
	Name        *string            `json:"name"`
	Type        *string            `json:"type"`
	Description *string            `json:"description"`
	Config      *map[string]interface{} `json:"config"`
	GroupID     *uuid.UUID         `json:"group_id"`
}

type AlertPayload struct {
	AlertNo         string     `json:"alert_no"`                   // unique date-time related id
	RuleID          uuid.UUID  `json:"rule_id"`
	RuleName        string     `json:"rule_name"`
	Severity        string     `json:"severity"`
	Status          string     `json:"status"`                    // firing, resolved
	Description     string     `json:"description"`
	Labels          string     `json:"labels"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	RenderedContent string     `json:"rendered_content,omitempty"` // when rule has template_id, content rendered from template
}
