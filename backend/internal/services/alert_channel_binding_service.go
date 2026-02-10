package services

import (
	"alert-center/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertChannelBindingService struct {
	db *pgxpool.Pool
}

func NewAlertChannelBindingService(db *pgxpool.Pool) *AlertChannelBindingService {
	return &AlertChannelBindingService{db: db}
}

func (s *AlertChannelBindingService) BindChannels(ctx context.Context, ruleID uuid.UUID, channelIDs []uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM alert_channel_bindings WHERE rule_id = $1`, ruleID)
	if err != nil {
		return err
	}

	for _, channelID := range channelIDs {
		binding := &models.AlertChannelBinding{
			ID:        uuid.New(),
			RuleID:    ruleID,
			ChannelID: channelID,
			Status:    1,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO alert_channel_bindings (id, rule_id, channel_id, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, binding.ID, binding.RuleID, binding.ChannelID, binding.Status)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *AlertChannelBindingService) GetByRuleID(ctx context.Context, ruleID uuid.UUID) ([]models.AlertChannel, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ac.id, ac.name, ac.type, ac.description, ac.config, ac.group_id, ac.status, ac.created_at, ac.updated_at
		FROM alert_channels ac
		INNER JOIN alert_channel_bindings acb ON ac.id = acb.channel_id
		WHERE acb.rule_id = $1 AND ac.status = 1
	`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.AlertChannel
	for rows.Next() {
		var ch models.AlertChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Description, &ch.Config,
			&ch.GroupID, &ch.Status, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}

	return channels, nil
}

// GetChannelsByRuleIDs returns bound channels (id, name, type) for the given rule IDs in one query.
// Result map: rule_id -> list of channels (minimal fields for display).
func (s *AlertChannelBindingService) GetChannelsByRuleIDs(ctx context.Context, ruleIDs []uuid.UUID) (map[uuid.UUID][]models.AlertChannel, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT acb.rule_id, ac.id, ac.name, ac.type
		FROM alert_channels ac
		INNER JOIN alert_channel_bindings acb ON ac.id = acb.channel_id
		WHERE acb.rule_id = ANY($1) AND ac.status = 1
	`, ruleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]models.AlertChannel)
	for rows.Next() {
		var ruleID, chID uuid.UUID
		var name, chType string
		if err := rows.Scan(&ruleID, &chID, &name, &chType); err != nil {
			return nil, err
		}
		out[ruleID] = append(out[ruleID], models.AlertChannel{ID: chID, Name: name, Type: chType})
	}
	return out, nil
}

func (s *AlertChannelBindingService) SendToBoundChannels(ctx context.Context, ruleID uuid.UUID, alert *AlertPayload) error {
	channels, err := s.GetByRuleID(ctx, ruleID)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		var config map[string]interface{}
		json.Unmarshal([]byte(channel.Config), &config)

		switch channel.Type {
		case "lark":
			_ = sendLarkAlert(ctx, config, alert)
		case "telegram":
			_ = sendTelegramAlert(ctx, config, alert)
		case "webhook":
			_ = sendWebhookAlert(ctx, config, alert)
		}
	}

	return nil
}

func sendLarkAlert(ctx context.Context, config map[string]interface{}, alert *AlertPayload) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok {
		return nil
	}
	payload := buildLarkCardPayload(alert)
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	return nil
}

func sendTelegramAlert(ctx context.Context, config map[string]interface{}, alert *AlertPayload) error {
	botToken, ok := config["bot_token"].(string)
	if !ok {
		return nil
	}
	chatID, ok := config["chat_id"].(string)
	if !ok {
		return nil
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
			text = fmt.Sprintf("✅ *告警恢复*\n\n*告警编号*: %s\n*规则*: %s\n*级别*: %s\n*状态*: %s\n*恢复时间*: %s\n*持续时长*: %s",
				alertNoStr, alert.RuleName, alert.Severity, alert.Status,
				alert.EndedAt.Format("2006-01-02 15:04:05"), dur.String())
		} else {
			text = fmt.Sprintf("🚨 *告警通知*\n\n*告警编号*: %s\n*规则*: %s\n*级别*: %s\n*状态*: %s",
				alertNoStr, alert.RuleName, alert.Severity, alert.Status)
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
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	return nil
}


func sendWebhookAlert(ctx context.Context, config map[string]interface{}, alert *AlertPayload) error {
	webhookURL, ok := config["url"].(string)
	if !ok {
		return nil
	}

	var body []byte
	if isLarkWebhookURL(webhookURL) {
		var content string
		if alert.RenderedContent != "" {
			if alert.Status == "resolved" {
				content = "**告警恢复**\n\n" + alert.RenderedContent
			} else {
				content = "**告警通知**\n\n" + alert.RenderedContent
			}
		} else {
			alertNoStr := alert.AlertNo
			if alertNoStr == "" {
				alertNoStr = "-"
			}
			if alert.Status == "resolved" && alert.EndedAt != nil {
				dur := alert.EndedAt.Sub(alert.StartedAt).Round(time.Second)
				content = fmt.Sprintf("**告警恢复**\n\n**告警编号**: %s\n**规则**: %s\n**级别**: %s\n**状态**: %s\n**开始时间**: %s\n**恢复时间**: %s\n**持续时长**: %s",
					alertNoStr, alert.RuleName, alert.Severity, alert.Status,
					alert.StartedAt.Format("2006-01-02 15:04:05"),
					alert.EndedAt.Format("2006-01-02 15:04:05"), dur.String())
			} else {
				content = fmt.Sprintf("**告警通知**\n\n**告警编号**: %s\n**规则**: %s\n**级别**: %s\n**状态**: %s\n**时间**: %s",
					alertNoStr, alert.RuleName, alert.Severity, alert.Status, alert.StartedAt.Format("2006-01-02 15:04:05"))
			}
		}
		payload := map[string]interface{}{
			"msg_type": "markdown",
			"content":  map[string]interface{}{"text": content},
		}
		body, _ = json.Marshal(payload)
	} else {
		body, _ = json.Marshal(alert)
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	return nil
}
