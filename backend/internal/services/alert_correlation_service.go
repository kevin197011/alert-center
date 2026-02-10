package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"alert-center/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertCorrelationService struct {
	db *pgxpool.Pool
}

func NewAlertCorrelationService(db *pgxpool.Pool) *AlertCorrelationService {
	return &AlertCorrelationService{db: db}
}

type CorrelatedAlert struct {
	RootCause        *models.AlertHistory   `json:"root_cause"`
	RelatedAlerts    []*models.AlertHistory `json:"related_alerts"`
	CorrelationScore float64                `json:"correlation_score"`
	CommonLabels     map[string]string      `json:"common_labels"`
	TimeWindow       time.Duration          `json:"time_window"`
}

type LabelSimilarity struct {
	Label      string
	Similarity float64
}

func (s *AlertCorrelationService) AnalyzeCorrelations(ctx context.Context, alertID uuid.UUID, timeWindow time.Duration) (*CorrelatedAlert, error) {
	alert, err := s.getAlertByID(ctx, alertID)
	if err != nil {
		return nil, err
	}

	relatedAlerts, err := s.getRelatedAlerts(ctx, alert, timeWindow)
	if err != nil {
		return nil, err
	}

	if len(relatedAlerts) == 0 {
		var labelsMap map[string]string
		json.Unmarshal([]byte(alert.Labels), &labelsMap)
		return &CorrelatedAlert{
			RootCause:     alert,
			RelatedAlerts: []*models.AlertHistory{},
			CommonLabels:  labelsMap,
			TimeWindow:    timeWindow,
		}, nil
	}

	rootCause := s.identifyRootCause(alert, relatedAlerts)
	commonLabels := s.findCommonLabels(alert, relatedAlerts)
	correlationScore := s.calculateCorrelationScore(alert, relatedAlerts, commonLabels)

	return &CorrelatedAlert{
		RootCause:        rootCause,
		RelatedAlerts:    relatedAlerts,
		CorrelationScore: correlationScore,
		CommonLabels:     commonLabels,
		TimeWindow:       timeWindow,
	}, nil
}

func (s *AlertCorrelationService) getAlertByID(ctx context.Context, id uuid.UUID) (*models.AlertHistory, error) {
	var alert models.AlertHistory
	err := s.db.QueryRow(ctx, `
		SELECT id, COALESCE(alert_no, ''), rule_id, fingerprint, severity, status, started_at, ended_at, labels, annotations, payload, created_at
		FROM alert_history WHERE id=$1
	`, id).Scan(&alert.ID, &alert.AlertNo, &alert.RuleID, &alert.Fingerprint, &alert.Severity, &alert.Status,
		&alert.StartedAt, &alert.EndedAt, &alert.Labels, &alert.Annotations, &alert.Payload, &alert.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *AlertCorrelationService) getRelatedAlerts(ctx context.Context, alert *models.AlertHistory, window time.Duration) ([]*models.AlertHistory, error) {
	startTime := alert.StartedAt.Add(-window)
	endTime := alert.StartedAt.Add(window)

	rows, err := s.db.Query(ctx, `
		SELECT id, COALESCE(alert_no, ''), rule_id, fingerprint, severity, status, started_at, ended_at, labels, annotations, payload, created_at
		FROM alert_history
		WHERE started_at BETWEEN $1 AND $2 AND id != $3
		ORDER BY started_at ASC
	`, startTime, endTime, alert.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.AlertHistory
	for rows.Next() {
		var a models.AlertHistory
		if err := rows.Scan(&a.ID, &a.AlertNo, &a.RuleID, &a.Fingerprint, &a.Severity, &a.Status,
			&a.StartedAt, &a.EndedAt, &a.Labels, &a.Annotations, &a.Payload, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, &a)
	}

	return alerts, nil
}

func (s *AlertCorrelationService) identifyRootCause(alert *models.AlertHistory, related []*models.AlertHistory) *models.AlertHistory {
	if len(related) == 0 {
		return alert
	}

	scores := make(map[uuid.UUID]float64)
	scores[alert.ID] = 0

	for _, relatedAlert := range related {
		scores[relatedAlert.ID] = 0

		similarity := s.calculateLabelSimilarity(alert.Labels, relatedAlert.Labels)
		timeDistance := math.Abs(float64(alert.StartedAt.Sub(relatedAlert.StartedAt).Milliseconds()))
		timeScore := 1.0 / (1.0 + timeDistance/60000)

		scores[relatedAlert.ID] = similarity*0.7 + timeScore*0.3
	}

	var rootCause *models.AlertHistory
	maxScore := float64(-1)
	for id, score := range scores {
		if score > maxScore {
			maxScore = score
			if id == alert.ID {
				rootCause = alert
			} else {
				for _, a := range related {
					if a.ID == id {
						rootCause = a
						break
					}
				}
			}
		}
	}

	return rootCause
}

func (s *AlertCorrelationService) calculateLabelSimilarity(labels1, labels2 string) float64 {
	var m1, m2 map[string]string
	json.Unmarshal([]byte(labels1), &m1)
	json.Unmarshal([]byte(labels2), &m2)

	if m1 == nil || m2 == nil {
		return 0
	}

	var common int
	for k := range m1 {
		if v, ok := m2[k]; ok && v == m1[k] {
			common++
		}
	}

	total := len(m1) + len(m2) - common
	if total == 0 {
		return 1
	}

	return float64(common) / float64(total)
}

func (s *AlertCorrelationService) findCommonLabels(alert *models.AlertHistory, related []*models.AlertHistory) map[string]string {
	allLabels := make(map[string]map[string]int)
	totalCount := len(related) + 1

	var m1 map[string]string
	json.Unmarshal([]byte(alert.Labels), &m1)
	for k, v := range m1 {
		allLabels[k] = map[string]int{v: 1}
	}

	for _, a := range related {
		var m2 map[string]string
		json.Unmarshal([]byte(a.Labels), &m2)
		for k, v := range m2 {
			if existing, ok := allLabels[k]; ok {
				existing[v]++
			} else {
				allLabels[k] = map[string]int{v: 1}
			}
		}
	}

	commonLabels := make(map[string]string)
	for k, v := range allLabels {
		for labelValue, count := range v {
			if float64(count)/float64(totalCount) > 0.5 {
				commonLabels[k] = labelValue
			}
		}
	}

	return commonLabels
}

func (s *AlertCorrelationService) calculateCorrelationScore(alert *models.AlertHistory, related []*models.AlertHistory, commonLabels map[string]string) float64 {
	if len(related) == 0 {
		return 0
	}

	var totalSimilarity float64
	for _, a := range related {
		similarity := s.calculateLabelSimilarity(alert.Labels, a.Labels)
		timeDistance := math.Abs(float64(alert.StartedAt.Sub(a.StartedAt).Milliseconds()))
		timeScore := 1.0 / (1.0 + timeDistance/300000)
		totalSimilarity += similarity*0.6 + timeScore*0.4
	}

	return totalSimilarity / float64(len(related))
}

func (s *AlertCorrelationService) FindPatterns(ctx context.Context, timeRange time.Duration, minOccurrences int) ([]AlertPattern, error) {
	startTime := time.Now().Add(-timeRange)

	rows, err := s.db.Query(ctx, `
		SELECT labels, COUNT(*) as count, array_agg(id) as ids
		FROM alert_history
		WHERE started_at >= $1
		GROUP BY labels
		HAVING COUNT(*) >= $2
		ORDER BY count DESC
	`, startTime, minOccurrences)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []AlertPattern
	for rows.Next() {
		var p AlertPattern
		var ids []uuid.UUID
		if err := rows.Scan(&p.CommonLabels, &p.OccurrenceCount, &ids); err != nil {
			return nil, err
		}
		p.AlertIDs = ids
		patterns = append(patterns, p)
	}

	return patterns, nil
}

type AlertPattern struct {
	CommonLabels    map[string]string `json:"common_labels"`
	OccurrenceCount int               `json:"occurrence_count"`
	AlertIDs        []uuid.UUID       `json:"alert_ids"`
	FirstSeen       time.Time         `json:"first_seen"`
	LastSeen        time.Time         `json:"last_seen"`
}

func (s *AlertCorrelationService) GroupSimilarAlerts(ctx context.Context, timeRange time.Duration, similarityThreshold float64) ([][]*models.AlertHistory, error) {
	startTime := time.Now().Add(-timeRange)

	rows, err := s.db.Query(ctx, `
		SELECT id, COALESCE(alert_no, ''), rule_id, fingerprint, severity, status, started_at, ended_at, labels, annotations, payload, created_at
		FROM alert_history
		WHERE started_at >= $1 AND status = 'firing'
	`, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allAlerts []*models.AlertHistory
	for rows.Next() {
		var a models.AlertHistory
		if err := rows.Scan(&a.ID, &a.AlertNo, &a.RuleID, &a.Fingerprint, &a.Severity, &a.Status,
			&a.StartedAt, &a.EndedAt, &a.Labels, &a.Annotations, &a.Payload, &a.CreatedAt); err != nil {
			return nil, err
		}
		allAlerts = append(allAlerts, &a)
	}

	groups := s.groupBySimilarity(allAlerts, similarityThreshold)

	return groups, nil
}

func (s *AlertCorrelationService) groupBySimilarity(alerts []*models.AlertHistory, threshold float64) [][]*models.AlertHistory {
	visited := make(map[int]bool)
	var groups [][]*models.AlertHistory

	for i := 0; i < len(alerts); i++ {
		if visited[i] {
			continue
		}

		var group []*models.AlertHistory
		group = append(group, alerts[i])
		visited[i] = true

		for j := i + 1; j < len(alerts); j++ {
			if visited[j] {
				continue
			}

			similarity := s.calculateLabelSimilarity(alerts[i].Labels, alerts[j].Labels)
			if similarity >= threshold {
				group = append(group, alerts[j])
				visited[j] = true
			}
		}

		if len(group) > 1 {
			sort.Slice(group, func(a, b int) bool {
				return group[a].StartedAt.Before(group[b].StartedAt)
			})
			groups = append(groups, group)
		}
	}

	return groups
}

func (s *AlertCorrelationService) PredictFutureAlerts(ctx context.Context, ruleID uuid.UUID, timeWindow time.Duration) ([]time.Time, error) {
	rows, err := s.db.Query(ctx, `
		SELECT started_at FROM alert_history
		WHERE rule_id = $1 AND status = 'firing'
		ORDER BY started_at DESC
		LIMIT 100
	`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var occurrences []time.Time
	for rows.Next() {
		var t time.Time
		rows.Scan(&t)
		occurrences = append(occurrences, t)
	}

	if len(occurrences) < 10 {
		return []time.Time{}, nil
	}

	intervals := make([]time.Duration, 0, len(occurrences)-1)
	for i := 0; i < len(occurrences)-1; i++ {
		intervals = append(intervals, occurrences[i].Sub(occurrences[i+1]))
	}

	var avgInterval time.Duration
	for _, d := range intervals {
		avgInterval += d
	}
	avgInterval /= time.Duration(len(intervals))

	var variance time.Duration
	for _, d := range intervals {
		diff := d - avgInterval
		variance += diff * diff
	}
	variance /= time.Duration(len(intervals))
	_ = time.Duration(math.Sqrt(float64(variance)))

	lastOccurrence := occurrences[0]
	nextOccurrences := make([]time.Time, 0, 3)
	currentTime := time.Now()

	for i := 1; i <= 3; i++ {
		nextTime := lastOccurrence.Add(avgInterval * time.Duration(i))
		if nextTime.After(currentTime) {
			nextOccurrences = append(nextOccurrences, nextTime)
		}
	}

	log.Printf("Predicted next occurrences for rule %s: %v", ruleID, nextOccurrences)
	return nextOccurrences, nil
}

type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	AlertID   uuid.UUID `json:"alert_id"`
	RuleName  string    `json:"rule_name"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	EventType string    `json:"event_type"` // firing, resolved, escalated
	Labels    string    `json:"labels"`
	RootCause bool      `json:"root_cause,omitempty"`
}

func (s *AlertCorrelationService) GenerateTimeline(ctx context.Context, fingerprint string, timeRange time.Duration) ([]TimelineEvent, error) {
	startTime := time.Now().Add(-timeRange)

	rows, err := s.db.Query(ctx, `
		SELECT h.id, h.started_at, h.ended_at, h.severity, h.status, h.labels, COALESCE(r.name, '')
		FROM alert_history h
		LEFT JOIN alert_rules r ON h.rule_id = r.id
		WHERE h.fingerprint = $1 AND h.started_at >= $2
		ORDER BY h.started_at ASC
	`, fingerprint, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		var resolvedAt *time.Time
		if err := rows.Scan(&e.AlertID, &e.Timestamp, &resolvedAt, &e.Severity, &e.Status, &e.Labels, &e.RuleName); err != nil {
			return nil, err
		}

		if e.Status == "firing" {
			e.EventType = "firing"
		} else {
			e.EventType = "resolved"
		}

		events = append(events, e)
	}

	return events, nil
}

func (s *AlertCorrelationService) DetectFlapping(ctx context.Context, ruleID uuid.UUID, window time.Duration, threshold int) ([]string, error) {
	startTime := time.Now().Add(-window)

	rows, err := s.db.Query(ctx, `
		SELECT id, started_at, ended_at FROM alert_history
		WHERE rule_id = $1 AND started_at >= $2
		ORDER BY started_at ASC
	`, ruleID, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []struct {
		ID    uuid.UUID
		Start time.Time
		End   *time.Time
	}

	for rows.Next() {
		var a struct {
			ID    uuid.UUID
			Start time.Time
			End   *time.Time
		}
		rows.Scan(&a.ID, &a.Start, &a.End)
		alerts = append(alerts, a)
	}

	var flappingPeriods []string
	var stateChanges []time.Time

	for _, alert := range alerts {
		if alert.End == nil {
			stateChanges = append(stateChanges, alert.Start)
		} else {
			stateChanges = append(stateChanges, alert.Start, *alert.End)
		}
	}

	sort.Slice(stateChanges, func(i, j int) bool {
		return stateChanges[i].Before(stateChanges[j])
	})

	for i := 1; i < len(stateChanges); i++ {
		duration := stateChanges[i].Sub(stateChanges[i-1])
		if duration < 5*time.Minute {
			flappingPeriods = append(flappingPeriods, fmt.Sprintf(
				"%s - %s (间隔: %v)",
				stateChanges[i-1].Format("15:04:05"),
				stateChanges[i].Format("15:04:05"),
				duration,
			))
		}
	}

	if len(flappingPeriods) >= threshold {
		log.Printf("Detected flapping for rule %s: %d state changes", ruleID, len(flappingPeriods))
	}

	return flappingPeriods, nil
}

func init() {
	log.Println("Alert correlation service initialized")
}
