package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Sample 查询结果样本
type Sample struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// QueryResult 查询结果
type QueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  Sample            `json:"value,omitempty"`
	Values []Sample          `json:"values,omitempty"`
}

// DataSource 监控数据源
type DataSource struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Type        string     `json:"type" gorm:"size:32;not null"` // prometheus, victoria-metrics
	Description string     `json:"description" gorm:"size:512"`
	Endpoint    string     `json:"endpoint" gorm:"size:512;not null"`
	Config      string     `json:"config" gorm:"type:jsonb"` // 额外配置
	Status      int        `json:"status" gorm:"default:1"`  // 0: disabled, 1: enabled
	HealthStatus string   `json:"health_status" gorm:"size:32;default:unknown"` // unknown, healthy, unhealthy
	LastCheckAt *time.Time `json:"last_check_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AlertSilence 告警静默规则
type AlertSilence struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Description string     `json:"description" gorm:"size:512"`
	Matchers    string     `json:"matchers" gorm:"type:jsonb"` // 标签匹配规则
	StartTime  time.Time  `json:"start_time" gorm:"not null"`
	EndTime    time.Time  `json:"end_time" gorm:"not null"`
	CreatedBy   uuid.UUID  `json:"created_by" gorm:"type:uuid"`
	Status      int        `json:"status" gorm:"default:1"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AlertSuppression 告警抑制规则
type AlertSuppression struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name         string     `json:"name" gorm:"size:128;not null"`
	Description  string     `json:"description" gorm:"size:512"`
	SourceMatcher string   `json:"source_matcher" gorm:"type:jsonb"` // 源标签
	TargetMatcher string   `json:"target_matcher" gorm:"type:jsonb"` // 目标标签
	Priority     int       `json:"priority" gorm:"default:0"` // 优先级
	Status       int       `json:"status" gorm:"default:1"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FiringAlert 正在触发的告警
type FiringAlert struct {
	RuleID      uuid.UUID            `json:"rule_id"`
	RuleName    string               `json:"rule_name"`
	Severity    string               `json:"severity"`
	Fingerprint string               `json:"fingerprint"`
	Labels      map[string]string    `json:"labels"`
	Annotations map[string]string    `json:"annotations"`
	StartsAt    time.Time           `json:"starts_at"`
	EndsAt      *time.Time          `json:"ends_at,omitempty"`
	Value       float64              `json:"value"`
	Status      string               `json:"status"` // firing, resolved
}

// GenerateFingerprint 生成告警指纹
func GenerateFingerprint(labels map[string]string) string {
	data, _ := json.Marshal(labels)
	return string(data)
}

// NotificationTemplate 通知模板
type NotificationTemplate struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Description string     `json:"description" gorm:"size:512"`
	Type        string     `json:"type" gorm:"size:32"` // markdown, text, html
	ChannelType string     `json:"channel_type" gorm:"size:32"` // lark, telegram, email, webhook
	Subject     string     `json:"subject" gorm:"size:256"`
	Content     string     `json:"content" gorm:"type:text"`
	Variables   string     `json:"variables" gorm:"type:jsonb"`
	Status      int        `json:"status" gorm:"default:1"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AlertEscalation 告警升级规则
type AlertEscalation struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name          string     `json:"name" gorm:"size:128;not null"`
	Description   string     `json:"description" gorm:"size:512"`
	RuleID        uuid.UUID  `json:"rule_id" gorm:"type:uuid"` // 关联的告警规则
	Severity      string     `json:"severity" gorm:"size:32"` // 原始级别
	EscalateTo    string     `json:"escalate_to" gorm:"size:32"` // 升级到的级别
	WaitMinutes   int        `json:"wait_minutes" gorm:"default:5"` // 等待分钟数后升级
	ChannelID     uuid.UUID  `json:"channel_id" gorm:"type:uuid"` // 升级通知渠道
	RepeatCount   int        `json:"repeat_count" gorm:"default:0"` // 重复通知次数，0表示不重复
	RepeatMinutes int        `json:"repeat_minutes" gorm:"default:30"` // 重复通知间隔
	Status        int        `json:"status" gorm:"default:1"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AlertEscalationLog 升级记录
type AlertEscalationLog struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	EscalationID  uuid.UUID  `json:"escalation_id" gorm:"type:uuid"`
	AlertID       uuid.UUID  `json:"alert_id" gorm:"type:uuid"` // alert_history.id
	FromSeverity  string     `json:"from_severity" gorm:"size:32"`
	ToSeverity    string     `json:"to_severity" gorm:"size:32"`
	ChannelID     uuid.UUID  `json:"channel_id" gorm:"type:uuid"`
	NotifiedAt    time.Time  `json:"notified_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
