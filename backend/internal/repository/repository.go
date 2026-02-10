package repository

import (
	"alert-center/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewDatabase() (*Database, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.name"),
		viper.GetString("database.sslmode"),
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	maxOpen := viper.GetInt("database.max_open_conns")
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := viper.GetInt("database.max_idle_conns")
	if maxIdle <= 0 {
		maxIdle = 5
	}
	maxLifetime := viper.GetInt("database.conn_max_lifetime")
	if maxLifetime <= 0 {
		maxLifetime = 300
	}
	config.MaxConns = int32(maxOpen)
	config.MinConns = int32(maxIdle)
	config.MaxConnLifetime = time.Duration(maxLifetime) * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

func (d *Database) Close() {
	d.Pool.Close()
}

// User Repository
type UserRepository struct {
	db *Database
}

func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO users (id, username, password, email, phone, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.Username, user.Password, user.Email, user.Phone, user.Role, user.Status, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, username, password, email, phone, role, status, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Phone,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, username, password, email, phone, role, status, created_at, updated_at, last_login_at
		FROM users WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Phone,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Pool.Exec(ctx, `UPDATE users SET last_login_at = $1 WHERE id = $2`, now, id)
	return err
}

// BusinessGroup Repository
type BusinessGroupRepository struct {
	db *Database
}

func NewBusinessGroupRepository(db *Database) *BusinessGroupRepository {
	return &BusinessGroupRepository{db: db}
}

func (r *BusinessGroupRepository) Create(ctx context.Context, group *models.BusinessGroup) error {
	group.ID = uuid.New()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO business_groups (id, name, description, parent_id, manager_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, group.ID, group.Name, group.Description, group.ParentID, group.ManagerID, group.Status, group.CreatedAt, group.UpdatedAt)
	return err
}

func (r *BusinessGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BusinessGroup, error) {
	var group models.BusinessGroup
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, parent_id, manager_id, status, created_at, updated_at
		FROM business_groups WHERE id = $1
	`, id).Scan(&group.ID, &group.Name, &group.Description, &group.ParentID,
		&group.ManagerID, &group.Status, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *BusinessGroupRepository) List(ctx context.Context, page, pageSize int, status int) ([]models.BusinessGroup, int, error) {
	offset := (page - 1) * pageSize

	var groups []models.BusinessGroup
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, description, parent_id, manager_id, status, created_at, updated_at
		FROM business_groups
		WHERE ($1 = -1 OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var group models.BusinessGroup
		if err := rows.Scan(&group.ID, &group.Name, &group.Description, &group.ParentID,
			&group.ManagerID, &group.Status, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, 0, err
		}
		groups = append(groups, group)
	}

	var total int
	r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM business_groups WHERE ($1 = -1 OR status = $1)`,
		status).Scan(&total)

	return groups, total, nil
}

// AlertRule Repository
type AlertRuleRepository struct {
	db *Database
}

func NewAlertRuleRepository(db *Database) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

func (r *AlertRuleRepository) Create(ctx context.Context, rule *models.AlertRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	effectiveStart := rule.EffectiveStartTime
	if effectiveStart == "" {
		effectiveStart = "00:00"
	}
	effectiveEnd := rule.EffectiveEndTime
	if effectiveEnd == "" {
		effectiveEnd = "23:59"
	}
	excl := rule.ExclusionWindows
	if excl == "" {
		excl = "[]"
	}
	evalInterval := rule.EvaluationIntervalSeconds
	if evalInterval <= 0 {
		evalInterval = 60
	}
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO alert_rules (id, name, description, expression, evaluation_interval_seconds, for_duration, severity,
			labels, annotations, template_id, group_id, data_source_type, data_source_url, status,
			effective_start_time, effective_end_time, exclusion_windows, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, rule.ID, rule.Name, rule.Description, rule.Expression, evalInterval, rule.ForDuration, rule.Severity,
		rule.Labels, rule.Annotations, rule.TemplateID, rule.GroupID, rule.DataSourceType,
		rule.DataSourceURL, rule.Status, effectiveStart, effectiveEnd, excl, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (r *AlertRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertRule, error) {
	var rule models.AlertRule
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, expression, COALESCE(evaluation_interval_seconds, 60), for_duration, severity, labels, annotations,
			template_id, group_id, data_source_type, data_source_url, status,
			COALESCE(effective_start_time, '00:00'), COALESCE(effective_end_time, '23:59'), COALESCE(exclusion_windows::text, '[]'),
			created_at, updated_at
		FROM alert_rules WHERE id = $1
	`, id).Scan(&rule.ID, &rule.Name, &rule.Description, &rule.Expression, &rule.EvaluationIntervalSeconds, &rule.ForDuration,
		&rule.Severity, &rule.Labels, &rule.Annotations, &rule.TemplateID, &rule.GroupID,
		&rule.DataSourceType, &rule.DataSourceURL, &rule.Status,
		&rule.EffectiveStartTime, &rule.EffectiveEndTime, &rule.ExclusionWindows, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *AlertRuleRepository) List(ctx context.Context, page, pageSize int, groupID *uuid.UUID, severity, status string) ([]models.AlertRule, int, error) {
	offset := (page - 1) * pageSize

	query := `
		SELECT id, name, description, expression, COALESCE(evaluation_interval_seconds, 60), for_duration, severity, labels, annotations,
			template_id, group_id, data_source_type, data_source_url, status,
			COALESCE(effective_start_time, '00:00'), COALESCE(effective_end_time, '23:59'), COALESCE(exclusion_windows::text, '[]'),
			created_at, updated_at
		FROM alert_rules
		WHERE ($1::uuid IS NULL OR group_id = $1)
			AND ($2 = '' OR severity = $2)
			AND ($3 = '' OR status::text = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`

	rows, err := r.db.Pool.Query(ctx, query, groupID, severity, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rules []models.AlertRule
	for rows.Next() {
		var rule models.AlertRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.Expression, &rule.EvaluationIntervalSeconds, &rule.ForDuration,
			&rule.Severity, &rule.Labels, &rule.Annotations, &rule.TemplateID, &rule.GroupID,
			&rule.DataSourceType, &rule.DataSourceURL, &rule.Status,
			&rule.EffectiveStartTime, &rule.EffectiveEndTime, &rule.ExclusionWindows, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, 0, err
		}
		rules = append(rules, rule)
	}

	var total int
	countQuery := `
		SELECT COUNT(*) FROM alert_rules
		WHERE ($1::uuid IS NULL OR group_id = $1)
			AND ($2 = '' OR severity = $2)
			AND ($3 = '' OR status::text = $3)
	`
	r.db.Pool.QueryRow(ctx, countQuery, groupID, severity, status).Scan(&total)

	return rules, total, nil
}

func (r *AlertRuleRepository) Update(ctx context.Context, rule *models.AlertRule) error {
	rule.UpdatedAt = time.Now()
	effectiveStart := rule.EffectiveStartTime
	if effectiveStart == "" {
		effectiveStart = "00:00"
	}
	effectiveEnd := rule.EffectiveEndTime
	if effectiveEnd == "" {
		effectiveEnd = "23:59"
	}
	excl := rule.ExclusionWindows
	if excl == "" {
		excl = "[]"
	}
	evalInterval := rule.EvaluationIntervalSeconds
	if evalInterval <= 0 {
		evalInterval = 60
	}
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE alert_rules SET name=$1, description=$2, expression=$3, evaluation_interval_seconds=$4, for_duration=$5,
			severity=$6, labels=$7, annotations=$8, template_id=$9, group_id=$10,
			data_source_type=$11, data_source_url=$12, status=$13,
			effective_start_time=$14, effective_end_time=$15, exclusion_windows=$16, updated_at=$17
		WHERE id=$18
	`, rule.Name, rule.Description, rule.Expression, evalInterval, rule.ForDuration, rule.Severity,
		rule.Labels, rule.Annotations, rule.TemplateID, rule.GroupID, rule.DataSourceType,
		rule.DataSourceURL, rule.Status, effectiveStart, effectiveEnd, excl, rule.UpdatedAt, rule.ID)
	return err
}

func (r *AlertRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	return err
}

// AlertChannel Repository
type AlertChannelRepository struct {
	db *Database
}

func NewAlertChannelRepository(db *Database) *AlertChannelRepository {
	return &AlertChannelRepository{db: db}
}

func (r *AlertChannelRepository) Create(ctx context.Context, channel *models.AlertChannel) error {
	channel.ID = uuid.New()
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO alert_channels (id, name, type, description, config, group_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, channel.ID, channel.Name, channel.Type, channel.Description, channel.Config,
		channel.GroupID, channel.Status, channel.CreatedAt, channel.UpdatedAt)
	return err
}

func (r *AlertChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertChannel, error) {
	var ch models.AlertChannel
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, type, description, config, group_id, status, created_at, updated_at
		FROM alert_channels WHERE id = $1
	`, id).Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Description, &ch.Config,
		&ch.GroupID, &ch.Status, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *AlertChannelRepository) Update(ctx context.Context, channel *models.AlertChannel) error {
	channel.UpdatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE alert_channels SET name=$1, type=$2, description=$3, config=$4, group_id=$5, status=$6, updated_at=$7
		WHERE id=$8
	`, channel.Name, channel.Type, channel.Description, channel.Config, channel.GroupID, channel.Status, channel.UpdatedAt, channel.ID)
	return err
}

func (r *AlertChannelRepository) List(ctx context.Context, page, pageSize int, channelType string, status int) ([]models.AlertChannel, int, error) {
	offset := (page - 1) * pageSize

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, type, description, config, group_id, status, created_at, updated_at
		FROM alert_channels
		WHERE ($1 = '' OR type = $1) AND ($2 = -1 OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, channelType, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var channels []models.AlertChannel
	for rows.Next() {
		var ch models.AlertChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Description, &ch.Config,
			&ch.GroupID, &ch.Status, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, 0, err
		}
		channels = append(channels, ch)
	}

	var total int
	r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alert_channels
		WHERE ($1 = '' OR type = $1) AND ($2 = -1 OR status = $2)
	`, channelType, status).Scan(&total)

	return channels, total, nil
}

// AlertHistory Repository
type AlertHistoryRepository struct {
	db *Database
}

func NewAlertHistoryRepository(db *Database) *AlertHistoryRepository {
	return &AlertHistoryRepository{db: db}
}

// alertNo generates a unique date-time related id: AL + YYYYMMDDHHmmss + 8 hex chars.
func alertNo() string {
	t := time.Now().Format("20060102150405")
	s := uuid.New().String()
	return "AL" + t + "-" + s[:8]
}

func (r *AlertHistoryRepository) Create(ctx context.Context, history *models.AlertHistory) error {
	history.ID = uuid.New()
	history.CreatedAt = time.Now()
	if history.AlertNo == "" {
		history.AlertNo = alertNo()
	}

	labels := history.Labels
	if labels == "" {
		labels = "{}"
	}
	annotations := history.Annotations
	if annotations == "" {
		annotations = "{}"
	}

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO alert_history (id, alert_no, rule_id, fingerprint, severity, status, started_at, ended_at, labels, annotations, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, history.ID, history.AlertNo, history.RuleID, history.Fingerprint, history.Severity, history.Status,
		history.StartedAt, history.EndedAt, labels, annotations, history.Payload, history.CreatedAt)
	return err
}

func (r *AlertHistoryRepository) List(ctx context.Context, page, pageSize int, ruleID *uuid.UUID, status string,
	startTime, endTime *time.Time) ([]models.AlertHistory, int, error) {

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	// Use sentinel times when nil so PostgreSQL gets typed params (avoids 42P08)
	startArg := startTime
	if startArg == nil {
		t := time.Time{}
		startArg = &t
	}
	endArg := endTime
	if endArg == nil {
		t := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
		endArg = &t
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, COALESCE(alert_no, ''), rule_id, fingerprint, severity, status, started_at, ended_at,
			COALESCE(labels::text, ''), COALESCE(annotations::text, ''), payload, created_at
		FROM alert_history
		WHERE ($1::uuid IS NULL OR rule_id = $1)
			AND ($2 = '' OR status = $2)
			AND (started_at >= $3 AND started_at <= $4)
		ORDER BY started_at DESC
		LIMIT $5 OFFSET $6
	`, ruleID, status, startArg, endArg, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var histories []models.AlertHistory
	for rows.Next() {
		var h models.AlertHistory
		if err := rows.Scan(&h.ID, &h.AlertNo, &h.RuleID, &h.Fingerprint, &h.Severity, &h.Status,
			&h.StartedAt, &h.EndedAt, &h.Labels, &h.Annotations, &h.Payload, &h.CreatedAt); err != nil {
			return nil, 0, err
		}
		histories = append(histories, h)
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alert_history
		WHERE ($1::uuid IS NULL OR rule_id = $1)
			AND ($2 = '' OR status = $2)
			AND (started_at >= $3 AND started_at <= $4)
	`, ruleID, status, startArg, endArg).Scan(&total); err != nil {
		return nil, 0, err
	}
	return histories, total, nil
}

// GetLatestFiringByRuleAndFingerprint returns the most recent alert_history row with status='firing' for the given rule and fingerprint.
func (r *AlertHistoryRepository) GetLatestFiringByRuleAndFingerprint(ctx context.Context, ruleID uuid.UUID, fingerprint string) (*models.AlertHistory, error) {
	var h models.AlertHistory
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, COALESCE(alert_no, ''), rule_id, fingerprint, severity, status, started_at, ended_at,
			COALESCE(labels::text, '{}'), COALESCE(annotations::text, '{}'), payload, created_at
		FROM alert_history
		WHERE rule_id = $1 AND fingerprint = $2 AND status = 'firing'
		ORDER BY started_at DESC
		LIMIT 1
	`, ruleID, fingerprint).Scan(&h.ID, &h.AlertNo, &h.RuleID, &h.Fingerprint, &h.Severity, &h.Status,
		&h.StartedAt, &h.EndedAt, &h.Labels, &h.Annotations, &h.Payload, &h.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// MarkResolvedByRuleAndFingerprint sets the latest firing record for (rule_id, fingerprint) to status='resolved' and ended_at.
func (r *AlertHistoryRepository) MarkResolvedByRuleAndFingerprint(ctx context.Context, ruleID uuid.UUID, fingerprint string, endedAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE alert_history SET status = 'resolved', ended_at = $1
		WHERE id = (
			SELECT id FROM alert_history
			WHERE rule_id = $2 AND fingerprint = $3 AND status = 'firing'
			ORDER BY started_at DESC
			LIMIT 1
		)
	`, endedAt, ruleID, fingerprint)
	return err
}

func (r *AlertHistoryRepository) GetStatistics(ctx context.Context, startTime, endTime *time.Time, groupID *uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'firing') as firing,
			COUNT(*) FILTER (WHERE status = 'resolved') as resolved,
			COUNT(*) FILTER (WHERE severity = 'critical') as critical,
			COUNT(*) FILTER (WHERE severity = 'warning') as warning,
			COUNT(*) FILTER (WHERE severity = 'info') as info
		FROM alert_history
		WHERE started_at >= $1 AND started_at <= $2
			AND ($3::uuid IS NULL OR rule_id IN (
				SELECT id FROM alert_rules WHERE group_id = $3
			))
	`

	var result struct {
		Total    int `db:"total"`
		Firing   int `db:"firing"`
		Resolved int `db:"resolved"`
		Critical int `db:"critical"`
		Warning  int `db:"warning"`
		Info     int `db:"info"`
	}

	err := r.db.Pool.QueryRow(ctx, query, startTime, endTime, groupID).Scan(
		&result.Total, &result.Firing, &result.Resolved, &result.Critical, &result.Warning, &result.Info)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":    result.Total,
		"firing":   result.Firing,
		"resolved": result.Resolved,
		"critical": result.Critical,
		"warning":  result.Warning,
		"info":     result.Info,
	}, nil
}

// SLA Config Repository
type SLAConfigRepository struct {
	db *Database
}

func NewSLAConfigRepository(db *Database) *SLAConfigRepository {
	return &SLAConfigRepository{db: db}
}

type SLAConfig struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	Name               string    `db:"name" json:"name"`
	Severity           string    `db:"severity" json:"severity"`
	ResponseTimeMins   int       `db:"response_time_mins" json:"response_time_mins"`
	ResolutionTimeMins int       `db:"resolution_time_mins" json:"resolution_time_mins"`
	Priority           int       `db:"priority" json:"priority"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

func (r *SLAConfigRepository) Create(ctx context.Context, config *SLAConfig) error {
	config.ID = uuid.New()
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO sla_configs (id, name, severity, response_time_mins, resolution_time_mins, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, config.ID, config.Name, config.Severity, config.ResponseTimeMins, config.ResolutionTimeMins, config.Priority, config.CreatedAt, config.UpdatedAt)
	return err
}

func (r *SLAConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*SLAConfig, error) {
	var config SLAConfig
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, severity, response_time_mins, resolution_time_mins, priority, created_at, updated_at
		FROM sla_configs WHERE id = $1
	`, id).Scan(&config.ID, &config.Name, &config.Severity, &config.ResponseTimeMins, &config.ResolutionTimeMins, &config.Priority, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *SLAConfigRepository) GetBySeverity(ctx context.Context, severity string) ([]SLAConfig, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, severity, response_time_mins, resolution_time_mins, priority, created_at, updated_at
		FROM sla_configs WHERE severity = $1 ORDER BY priority DESC
	`, severity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []SLAConfig
	for rows.Next() {
		var config SLAConfig
		if err := rows.Scan(&config.ID, &config.Name, &config.Severity, &config.ResponseTimeMins, &config.ResolutionTimeMins, &config.Priority, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func (r *SLAConfigRepository) List(ctx context.Context) ([]SLAConfig, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, severity, response_time_mins, resolution_time_mins, priority, created_at, updated_at
		FROM sla_configs ORDER BY priority DESC, severity ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []SLAConfig
	for rows.Next() {
		var config SLAConfig
		if err := rows.Scan(&config.ID, &config.Name, &config.Severity, &config.ResponseTimeMins, &config.ResolutionTimeMins, &config.Priority, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func (r *SLAConfigRepository) Update(ctx context.Context, config *SLAConfig) error {
	config.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		UPDATE sla_configs SET name=$1, severity=$2, response_time_mins=$3, resolution_time_mins=$4, priority=$5, updated_at=$6
		WHERE id=$7
	`, config.Name, config.Severity, config.ResponseTimeMins, config.ResolutionTimeMins, config.Priority, config.UpdatedAt, config.ID)
	return err
}

func (r *SLAConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM sla_configs WHERE id=$1`, id)
	return err
}

// OnCall Schedule Repository
type OnCallScheduleRepository struct {
	db *Database
}

func NewOnCallScheduleRepository(db *Database) *OnCallScheduleRepository {
	return &OnCallScheduleRepository{db: db}
}

type OnCallSchedule struct {
	ID            uuid.UUID `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Description   string    `db:"description" json:"description"`
	Timezone      string    `db:"timezone" json:"timezone"`
	RotationType  string    `db:"rotation_type" json:"rotation_type"`
	RotationStart time.Time `db:"rotation_start" json:"rotation_start"`
	Enabled       bool      `db:"enabled" json:"enabled"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

func (r *OnCallScheduleRepository) Create(ctx context.Context, schedule *OnCallSchedule) error {
	schedule.ID = uuid.New()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO oncall_schedules (id, name, description, timezone, rotation_type, rotation_start, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, schedule.ID, schedule.Name, schedule.Description, schedule.Timezone, schedule.RotationType, schedule.RotationStart, schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
	return err
}

func (r *OnCallScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*OnCallSchedule, error) {
	var schedule OnCallSchedule
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, timezone, rotation_type, rotation_start, enabled, created_at, updated_at
		FROM oncall_schedules WHERE id = $1
	`, id).Scan(&schedule.ID, &schedule.Name, &schedule.Description, &schedule.Timezone, &schedule.RotationType, &schedule.RotationStart, &schedule.Enabled, &schedule.CreatedAt, &schedule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *OnCallScheduleRepository) List(ctx context.Context) ([]OnCallSchedule, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, description, timezone, rotation_type, rotation_start, enabled, created_at, updated_at
		FROM oncall_schedules ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []OnCallSchedule
	for rows.Next() {
		var schedule OnCallSchedule
		if err := rows.Scan(&schedule.ID, &schedule.Name, &schedule.Description, &schedule.Timezone, &schedule.RotationType, &schedule.RotationStart, &schedule.Enabled, &schedule.CreatedAt, &schedule.UpdatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

func (r *OnCallScheduleRepository) Update(ctx context.Context, schedule *OnCallSchedule) error {
	schedule.UpdatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		UPDATE oncall_schedules SET name=$1, description=$2, timezone=$3, rotation_type=$4, rotation_start=$5, enabled=$6, updated_at=$7
		WHERE id=$8
	`, schedule.Name, schedule.Description, schedule.Timezone, schedule.RotationType, schedule.RotationStart, schedule.Enabled, schedule.UpdatedAt, schedule.ID)
	return err
}

func (r *OnCallScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM oncall_schedules WHERE id=$1`, id)
	return err
}

// OnCall Member Repository
type OnCallMemberRepository struct {
	db *Database
}

func NewOnCallMemberRepository(db *Database) *OnCallMemberRepository {
	return &OnCallMemberRepository{db: db}
}

type OnCallMember struct {
	ID         uuid.UUID `db:"id" json:"id"`
	ScheduleID uuid.UUID `db:"schedule_id" json:"schedule_id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Username   string    `db:"username" json:"username"`
	Email      string    `db:"email" json:"email"`
	Phone      string    `db:"phone" json:"phone"`
	Priority   int       `db:"priority" json:"priority"`
	StartTime  time.Time `db:"start_time" json:"start_time"`
	EndTime    time.Time `db:"end_time" json:"end_time"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (r *OnCallMemberRepository) Create(ctx context.Context, member *OnCallMember) error {
	member.ID = uuid.New()
	member.CreatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO oncall_members (id, schedule_id, user_id, username, email, phone, priority, start_time, end_time, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, member.ID, member.ScheduleID, member.UserID, member.Username, member.Email, member.Phone, member.Priority, member.StartTime, member.EndTime, member.IsActive, member.CreatedAt)
	return err
}

func (r *OnCallMemberRepository) GetByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]OnCallMember, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, schedule_id, user_id, username, email, phone, priority, start_time, end_time, is_active, created_at
		FROM oncall_members WHERE schedule_id=$1 AND is_active=true ORDER BY priority DESC
	`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []OnCallMember
	for rows.Next() {
		var member OnCallMember
		if err := rows.Scan(&member.ID, &member.ScheduleID, &member.UserID, &member.Username, &member.Email, &member.Phone, &member.Priority, &member.StartTime, &member.EndTime, &member.IsActive, &member.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}

func (r *OnCallMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM oncall_members WHERE id=$1`, id)
	return err
}

// OnCall Assignment Repository
type OnCallAssignmentRepository struct {
	db *Database
}

func NewOnCallAssignmentRepository(db *Database) *OnCallAssignmentRepository {
	return &OnCallAssignmentRepository{db: db}
}

type OnCallAssignment struct {
	ID         uuid.UUID `db:"id" json:"id"`
	ScheduleID uuid.UUID `db:"schedule_id" json:"schedule_id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Username   string    `db:"username" json:"username"`
	StartTime  time.Time `db:"start_time" json:"start_time"`
	EndTime    time.Time `db:"end_time" json:"end_time"`
	Email      string    `db:"email" json:"email"`
	Phone      string    `db:"phone" json:"phone"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (r *OnCallAssignmentRepository) Create(ctx context.Context, assignment *OnCallAssignment) error {
	assignment.ID = uuid.New()
	assignment.CreatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO oncall_assignments (id, schedule_id, user_id, username, start_time, end_time, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, assignment.ID, assignment.ScheduleID, assignment.UserID, assignment.Username, assignment.StartTime, assignment.EndTime, assignment.CreatedAt)
	return err
}

func (r *OnCallAssignmentRepository) GetCurrentByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*OnCallAssignment, error) {
	var assignment OnCallAssignment
	now := time.Now()
	err := r.db.Pool.QueryRow(ctx, `
		SELECT a.id, a.schedule_id, a.user_id, a.username, a.start_time, a.end_time, u.email, COALESCE(m.phone, ''), a.created_at
		FROM oncall_assignments a
		LEFT JOIN users u ON a.user_id = u.id
		LEFT JOIN oncall_members m ON a.schedule_id = m.schedule_id AND a.user_id = m.user_id
		WHERE a.schedule_id = $1 AND a.start_time <= $2 AND a.end_time > $2
	`, scheduleID, now).Scan(&assignment.ID, &assignment.ScheduleID, &assignment.UserID, &assignment.Username, &assignment.StartTime, &assignment.EndTime, &assignment.Email, &assignment.Phone, &assignment.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (r *OnCallAssignmentRepository) GetByScheduleID(ctx context.Context, scheduleID uuid.UUID, startTime, endTime time.Time) ([]OnCallAssignment, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT a.id, a.schedule_id, a.user_id, a.username, a.start_time, a.end_time, u.email, COALESCE(m.phone, ''), a.created_at
		FROM oncall_assignments a
		LEFT JOIN users u ON a.user_id = u.id
		LEFT JOIN oncall_members m ON a.schedule_id = m.schedule_id AND a.user_id = m.user_id
		WHERE a.schedule_id = $1 AND a.start_time < $2 AND a.end_time > $3
		ORDER BY a.start_time ASC
	`, scheduleID, endTime, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []OnCallAssignment
	for rows.Next() {
		var assignment OnCallAssignment
		if err := rows.Scan(&assignment.ID, &assignment.ScheduleID, &assignment.UserID, &assignment.Username, &assignment.StartTime, &assignment.EndTime, &assignment.Email, &assignment.Phone, &assignment.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

// Alert SLA Repository (for tracking per-alert SLA)
type AlertSLARepository struct {
	db *Database
}

func NewAlertSLARepository(db *Database) *AlertSLARepository {
	return &AlertSLARepository{db: db}
}

type AlertSLA struct {
	AlertID            uuid.UUID  `db:"alert_id"`
	RuleID             uuid.UUID  `db:"rule_id"`
	Severity           string     `db:"severity"`
	SLAConfigID        uuid.UUID  `db:"sla_config_id"`
	ResponseDeadline   *time.Time `db:"response_deadline"`
	ResolutionDeadline *time.Time `db:"resolution_deadline"`
	FirstAckedAt       *time.Time `db:"first_acked_at"`
	ResolvedAt         *time.Time `db:"resolved_at"`
	Status             string     `db:"status"`
	ResponseBreached   bool       `db:"response_breached"`
	ResolutionBreached bool       `db:"resolution_breached"`
	ResponseTimeSecs   float64    `db:"response_time_secs"`
	ResolutionTimeSecs float64    `db:"resolution_time_secs"`
	CreatedAt          time.Time  `db:"created_at"`
}

func (r *AlertSLARepository) Create(ctx context.Context, sla *AlertSLA) error {
	sla.CreatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO alert_slas (alert_id, rule_id, severity, sla_config_id, response_deadline, resolution_deadline, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, sla.AlertID, sla.RuleID, sla.Severity, sla.SLAConfigID, sla.ResponseDeadline, sla.ResolutionDeadline, sla.Status, sla.CreatedAt)
	return err
}

func (r *AlertSLARepository) GetByAlertID(ctx context.Context, alertID uuid.UUID) (*AlertSLA, error) {
	var sla AlertSLA
	err := r.db.Pool.QueryRow(ctx, `
		SELECT alert_id, rule_id, severity, sla_config_id, response_deadline, resolution_deadline,
		first_acked_at, resolved_at, status, response_breached, resolution_breached,
		COALESCE(response_time_secs, 0), COALESCE(resolution_time_secs, 0), created_at
		FROM alert_slas WHERE alert_id=$1
	`, alertID).Scan(&sla.AlertID, &sla.RuleID, &sla.Severity, &sla.SLAConfigID, &sla.ResponseDeadline, &sla.ResolutionDeadline,
		&sla.FirstAckedAt, &sla.ResolvedAt, &sla.Status, &sla.ResponseBreached, &sla.ResolutionBreached,
		&sla.ResponseTimeSecs, &sla.ResolutionTimeSecs, &sla.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sla, nil
}

func (r *AlertSLARepository) Update(ctx context.Context, sla *AlertSLA) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE alert_slas SET first_acked_at=$1, resolved_at=$2, status=$3,
		response_breached=$4, resolution_breached=$5, response_time_secs=$6, resolution_time_secs=$7
		WHERE alert_id=$8
	`, sla.FirstAckedAt, sla.ResolvedAt, sla.Status, sla.ResponseBreached, sla.ResolutionBreached,
		sla.ResponseTimeSecs, sla.ResolutionTimeSecs, sla.AlertID)
	return err
}
