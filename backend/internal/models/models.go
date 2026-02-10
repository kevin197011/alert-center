package models

import (
	"github.com/google/uuid"
	"time"
)

// User 用户模型
type User struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Password     string     `json:"-" gorm:"size:255;not null"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:128"`
	Phone        string     `json:"phone" gorm:"size:32"`
	Role         string     `json:"role" gorm:"size:32;default:user"`  // admin, manager, user
	Status       int        `json:"status" gorm:"default:1"`  // 0: disabled, 1: enabled
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// BusinessGroup 业务组
type BusinessGroup struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Description string     `json:"description" gorm:"size:512"`
	ParentID    *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	ManagerID   *uuid.UUID `json:"manager_id" gorm:"type:uuid"`
	Status      int        `json:"status" gorm:"default:1"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AlertChannel 告警渠道
type AlertChannel struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Type        string     `json:"type" gorm:"size:32;not null"`  // lark, telegram, email, webhook
	Description string     `json:"description" gorm:"size:512"`
	Config      string     `json:"config" gorm:"type:jsonb"`  // JSON配置
	GroupID     *uuid.UUID `json:"group_id" gorm:"type:uuid"`  // 所属业务组
	Status      int        `json:"status" gorm:"default:1"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AlertTemplate 告警模板
type AlertTemplate struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Description string     `json:"description" gorm:"size:512"`
	Content     string     `json:"content" gorm:"type:text;not null"`  // 模板内容
	Variables   string     `json:"variables" gorm:"type:jsonb"`  // 模板变量定义
	Type        string     `json:"type" gorm:"size:32;default:markdown"`  // markdown, text, html
	GroupID     *uuid.UUID `json:"group_id" gorm:"type:uuid"`
	Status      int        `json:"status" gorm:"default:1"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ExclusionWindow defines a time range when the rule must not fire. Days: 0=Sunday .. 6=Saturday; empty = every day.
type ExclusionWindow struct {
	Start string  `json:"start"` // HH:MM
	End   string  `json:"end"`   // HH:MM
	Days  []int   `json:"days"`  // 0-6, empty means all days
}

// AlertRule 告警规则
type AlertRule struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	Name               string     `json:"name" gorm:"size:128;not null"`
	Description        string     `json:"description" gorm:"size:512"`
	Expression               string     `json:"expression" gorm:"type:text;not null"`       // PromQL表达式
	EvaluationIntervalSeconds int        `json:"evaluation_interval_seconds" gorm:"default:60"` // 执行频率(秒)，规则评估间隔
	ForDuration              int        `json:"for_duration" gorm:"default:60"`             // 持续时间(秒)
	Severity                 string     `json:"severity" gorm:"size:32;not null"`           // critical, warning, info
	Labels             string     `json:"labels" gorm:"type:jsonb"`                // 告警标签
	Annotations        string     `json:"annotations" gorm:"type:jsonb"`           // 告警注释
	TemplateID         *uuid.UUID `json:"template_id" gorm:"type:uuid"`           // 关联模板
	GroupID            uuid.UUID  `json:"group_id" gorm:"type:uuid;not null"`      // 所属业务组
	DataSourceType     string     `json:"data_source_type" gorm:"size:32;default:prometheus"`
	DataSourceURL      string     `json:"data_source_url" gorm:"size:512"`
	Status             int        `json:"status" gorm:"default:1"`                    // 0: disabled, 1: enabled
	EffectiveStartTime string     `json:"effective_start_time" gorm:"size:5;default:00:00"` // 生效开始时间(每日), HH:MM, default 24h
	EffectiveEndTime   string     `json:"effective_end_time" gorm:"size:5;default:23:59"`   // 生效结束时间(每日), HH:MM
	ExclusionWindows   string     `json:"exclusion_windows" gorm:"type:jsonb"`              // 排除时间 JSON array of ExclusionWindow
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// AlertChannelBinding 告警渠道绑定
type AlertChannelBinding struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	RuleID     uuid.UUID  `json:"rule_id" gorm:"type:uuid;not null"`
	ChannelID  uuid.UUID  `json:"channel_id" gorm:"type:uuid;not null"`
	Status     int        `json:"status" gorm:"default:1"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AlertHistory 告警历史
type AlertHistory struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	AlertNo     string     `json:"alert_no" gorm:"size:32;uniqueIndex"` // unique date-time related id, e.g. AL20250205143022-a1b2c3
	RuleID      uuid.UUID  `json:"rule_id" gorm:"type:uuid;not null"`
	Fingerprint string     `json:"fingerprint" gorm:"size:256;index"`
	Severity    string     `json:"severity" gorm:"size:32"`
	Status      string     `json:"status" gorm:"size:32"`  // firing, resolved
	StartedAt   time.Time  `json:"started_at" gorm:"not null"`
	EndedAt     *time.Time `json:"ended_at"`
	Labels      string     `json:"labels" gorm:"type:jsonb"`
	Annotations  string     `json:"annotations" gorm:"type:jsonb"`
	Payload     string     `json:"payload" gorm:"type:text"`  // 原始告警数据
	CreatedAt   time.Time  `json:"created_at"`
}

// OperationLog 操作日志
type OperationLog struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid"`
	Action     string     `json:"action" gorm:"size:64"`
	Resource   string     `json:"resource" gorm:"size:128"`
	ResourceID string     `json:"resource_id" gorm:"size:128"`
	Detail     string     `json:"detail" gorm:"type:text"`
	IP        string     `json:"ip" gorm:"size:64"`
	CreatedAt  time.Time  `json:"created_at"`
}
