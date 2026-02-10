package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertStatisticsService struct {
	db *pgxpool.Pool
}

func NewAlertStatisticsService(db *pgxpool.Pool) *AlertStatisticsService {
	return &AlertStatisticsService{db: db}
}

type AlertStatistics struct {
	TotalAlerts      int64              `json:"total_alerts"`
	FiringAlerts     int64              `json:"firing_alerts"`
	ResolvedAlerts  int64              `json:"resolved_alerts"`
	CriticalAlerts  int64              `json:"critical_alerts"`
	WarningAlerts   int64              `json:"warning_alerts"`
	InfoAlerts      int64              `json:"info_alerts"`
	AvgResolveTime  float64            `json:"avg_resolve_time"` // 分钟
	BySeverity      []SeverityStats    `json:"by_severity"`
	ByStatus        []StatusStats     `json:"by_status"`
	ByDay           []DailyStats      `json:"by_day"`
	TopFiringRules  []RuleStats       `json:"top_firing_rules"`
}

type SeverityStats struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

type StatusStats struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type DailyStats struct {
	Date        string `json:"date"`
	Total       int64  `json:"total"`
	Firing      int64  `json:"firing"`
	Resolved    int64  `json:"resolved"`
	Critical    int64  `json:"critical"`
	Warning    int64  `json:"warning"`
}

type RuleStats struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	AlertCount  int64  `json:"alert_count"`
}

func (s *AlertStatisticsService) GetStatistics(ctx context.Context, startTime, endTime *time.Time, groupID *string) (*AlertStatistics, error) {
	stats := &AlertStatistics{}

	// Total alerts
	var totalQuery string
	var args []interface{}
	
	if groupID != nil && *groupID != "" {
		totalQuery = `
			SELECT COUNT(*) FROM alert_history ah
			INNER JOIN alert_rules ar ON ah.rule_id = ar.id
			WHERE ar.group_id = $1
		`
		args = []interface{}{*groupID}
	} else {
		totalQuery = `SELECT COUNT(*) FROM alert_history`
	}

	if startTime != nil {
		totalQuery += " AND ah.started_at >= $2"
		args = append(args, *startTime)
	}
	if endTime != nil {
		totalQuery += " AND ah.started_at <= $3"
		args = append(args, *endTime)
	}

	s.db.QueryRow(ctx, totalQuery, args...).Scan(&stats.TotalAlerts)

	// Firing alerts
	firingQuery := `SELECT COUNT(*) FROM alert_history WHERE status = 'firing'`
	if startTime != nil {
		firingQuery += " AND started_at >= $1"
		args = append(args, *startTime)
	}
	if endTime != nil {
		firingQuery += " AND started_at <= $2"
		args = append(args, *endTime)
	}
	s.db.QueryRow(ctx, firingQuery, args...).Scan(&stats.FiringAlerts)

	// Resolved alerts
	resolvedQuery := `SELECT COUNT(*) FROM alert_history WHERE status = 'resolved'`
	if startTime != nil {
		resolvedQuery += " AND ended_at >= $1"
		args = append(args, *startTime)
	}
	if endTime != nil {
		resolvedQuery += " AND ended_at <= $2"
		args = append(args, *endTime)
	}
	s.db.QueryRow(ctx, resolvedQuery, args...).Scan(&stats.ResolvedAlerts)

	// By severity
	severityRows, _ := s.db.Query(ctx, `
		SELECT severity, COUNT(*) FROM alert_history
		WHERE ($1::timestamp IS NULL OR started_at >= $1)
			AND ($2::timestamp IS NULL OR started_at <= $2)
		GROUP BY severity
	`, startTime, endTime)
	defer severityRows.Close()
	for severityRows.Next() {
		var s SeverityStats
		severityRows.Scan(&s.Severity, &s.Count)
		stats.BySeverity = append(stats.BySeverity, s)
		if s.Severity == "critical" {
			stats.CriticalAlerts = s.Count
		} else if s.Severity == "warning" {
			stats.WarningAlerts = s.Count
		} else if s.Severity == "info" {
			stats.InfoAlerts = s.Count
		}
	}

	// By status
	statusRows, _ := s.db.Query(ctx, `
		SELECT status, COUNT(*) FROM alert_history
		WHERE ($1::timestamp IS NULL OR started_at >= $1)
			AND ($2::timestamp IS NULL OR started_at <= $2)
		GROUP BY status
	`, startTime, endTime)
	defer statusRows.Close()
	for statusRows.Next() {
		var s StatusStats
		statusRows.Scan(&s.Status, &s.Count)
		stats.ByStatus = append(stats.ByStatus, s)
	}

	// By day (last 7 days)
	dayRows, _ := s.db.Query(ctx, `
		SELECT 
			DATE(started_at) as date,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'firing') as firing,
			COUNT(*) FILTER (WHERE status = 'resolved') as resolved,
			COUNT(*) FILTER (WHERE severity = 'critical') as critical,
			COUNT(*) FILTER (WHERE severity = 'warning') as warning
		FROM alert_history
		WHERE started_at >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY DATE(started_at)
		ORDER BY date DESC
	`)
	defer dayRows.Close()
	for dayRows.Next() {
		var d DailyStats
		dayRows.Scan(&d.Date, &d.Total, &d.Firing, &d.Resolved, &d.Critical, &d.Warning)
		stats.ByDay = append(stats.ByDay, d)
	}

	// Top firing rules
	ruleRows, _ := s.db.Query(ctx, `
		SELECT ah.rule_id, ar.name, COUNT(*) as count
		FROM alert_history ah
		INNER JOIN alert_rules ar ON ah.rule_id = ar.id
		WHERE ah.status = 'firing'
			AND ($1::timestamp IS NULL OR ah.started_at >= $1)
			AND ($2::timestamp IS NULL OR ah.started_at <= $2)
		GROUP BY ah.rule_id, ar.name
		ORDER BY count DESC
		LIMIT 10
	`, startTime, endTime)
	defer ruleRows.Close()
	for ruleRows.Next() {
		var r RuleStats
		ruleRows.Scan(&r.RuleID, &r.RuleName, &r.AlertCount)
		stats.TopFiringRules = append(stats.TopFiringRules, r)
	}

	return stats, nil
}

type DashboardSummary struct {
	TotalRules       int `json:"total_rules"`
	EnabledRules    int `json:"enabled_rules"`
	TotalChannels   int `json:"total_channels"`
	EnabledChannels int `json:"enabled_channels"`
	TodayAlerts    int `json:"today_alerts"`
	FiringAlerts    int `json:"firing_alerts"`
}

func (s *AlertStatisticsService) GetDashboardSummary(ctx context.Context) (*DashboardSummary, error) {
	summary := &DashboardSummary{}

	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_rules`).Scan(&summary.TotalRules)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_rules WHERE status = 1`).Scan(&summary.EnabledRules)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_channels`).Scan(&summary.TotalChannels)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_channels WHERE status = 1`).Scan(&summary.EnabledChannels)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_history WHERE DATE(started_at) = CURRENT_DATE`).Scan(&summary.TodayAlerts)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM alert_history WHERE status = 'firing'`).Scan(&summary.FiringAlerts)

	return summary, nil
}
