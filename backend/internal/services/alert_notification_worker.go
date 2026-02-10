package services

import (
	"alert-center/internal/models"
	"alert-center/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// formatMapToKeyValueLines parses jsonStr as a JSON object and returns markdown-style lines "**key**: value" per entry (keys sorted for stable output). Auto-adapts to any Prometheus labels/annotations.
func formatMapToKeyValueLines(jsonStr string) string {
	if jsonStr == "" || jsonStr == "{}" {
		return "-"
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return jsonStr
	}
	if len(m) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := m[k]
		vs := ""
		if v != nil {
			vs = fmt.Sprintf("%v", v)
		}
		b.WriteString("**")
		b.WriteString(k)
		b.WriteString("**: ")
		b.WriteString(vs)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// pendingKey identifies an alert that is currently satisfying the rule (for for_duration).
type pendingKey struct {
	ruleID      uuid.UUID
	fingerprint string
}

// pendingState tracks first-seen time and whether we have already sent notification for this firing period.
type pendingState struct {
	firstSeenAt time.Time
	notified    bool
}

// AlertNotificationWorker evaluates alert rules periodically and sends notifications.
type AlertNotificationWorker struct {
	db             *pgxpool.Pool
	ruleRepo       *repository.AlertRuleRepository
	historyRepo    *repository.AlertHistoryRepository
	evaluator      *AlertEvaluator
	sender         *NotificationSender
	templateSvc    *AlertTemplateService
	silenceSvc     *AlertSilenceService
	slaSvc         *SLAService
	slaBreachSvc   *SLABreachService
	broadcaster    Broadcaster
	checkInterval  time.Duration
	pendingMu      sync.Mutex
	pending        map[pendingKey]pendingState
}

// NewAlertNotificationWorker returns a new AlertNotificationWorker.
func NewAlertNotificationWorker(
	db *pgxpool.Pool,
	ruleRepo *repository.AlertRuleRepository,
	historyRepo *repository.AlertHistoryRepository,
	evaluator *AlertEvaluator,
	sender *NotificationSender,
	templateSvc *AlertTemplateService,
	silenceSvc *AlertSilenceService,
	slaSvc *SLAService,
	slaBreachSvc *SLABreachService,
	broadcaster Broadcaster,
	checkInterval time.Duration,
) *AlertNotificationWorker {
	return &AlertNotificationWorker{
		db:            db,
		ruleRepo:      ruleRepo,
		historyRepo:   historyRepo,
		evaluator:     evaluator,
		sender:        sender,
		templateSvc:   templateSvc,
		silenceSvc:    silenceSvc,
		slaSvc:        slaSvc,
		slaBreachSvc:  slaBreachSvc,
		broadcaster:   broadcaster,
		checkInterval: checkInterval,
		pending:       make(map[pendingKey]pendingState),
	}
}

// inEffectiveWindow returns true if t (server local) is within the rule's daily effective window.
func inEffectiveWindow(rule models.AlertRule, t time.Time) bool {
	start := rule.EffectiveStartTime
	end := rule.EffectiveEndTime
	if start == "" {
		start = "00:00"
	}
	if end == "" {
		end = "23:59"
	}
	nowMinutes := t.Hour()*60 + t.Minute()
	startMinutes := parseHHMM(start)
	endMinutes := parseHHMM(end)
	if startMinutes <= endMinutes {
		return nowMinutes >= startMinutes && nowMinutes <= endMinutes
	}
	// e.g. 22:00-06:00 spans midnight
	return nowMinutes >= startMinutes || nowMinutes <= endMinutes
}

func parseHHMM(s string) int {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

// inExclusionWindow returns true if t falls inside any of the rule's exclusion windows.
func inExclusionWindow(rule models.AlertRule, t time.Time) bool {
	if rule.ExclusionWindows == "" {
		return false
	}
	var windows []models.ExclusionWindow
	if err := json.Unmarshal([]byte(rule.ExclusionWindows), &windows); err != nil {
		return false
	}
	weekday := int(t.Weekday()) // 0=Sunday, 6=Saturday
	nowMinutes := t.Hour()*60 + t.Minute()
	for _, w := range windows {
		startM := parseHHMM(w.Start)
		endM := parseHHMM(w.End)
		// Check day: if Days is empty, applies every day
		if len(w.Days) > 0 {
			found := false
			for _, d := range w.Days {
				if d == weekday {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if startM <= endM {
			if nowMinutes >= startM && nowMinutes <= endM {
				return true
			}
		} else {
			if nowMinutes >= startM || nowMinutes <= endM {
				return true
			}
		}
	}
	return false
}

// Start runs the worker loop until ctx is cancelled.
func (w *AlertNotificationWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.runOnce(ctx); err != nil {
				log.Printf("AlertNotificationWorker runOnce: %v", err)
			}
		}
	}
}

func (w *AlertNotificationWorker) runOnce(ctx context.Context) error {
	// List enabled rules (status "1"); use a large page size to evaluate all.
	rules, _, err := w.ruleRepo.List(ctx, 1, 5000, nil, "", "1")
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	// Build minimal data source from rule (evaluator uses Endpoint and creates client on demand).
	seenThisRun := make(map[pendingKey]struct{})
	ruleByID := make(map[uuid.UUID]models.AlertRule)
	for _, rule := range rules {
		ruleByID[rule.ID] = rule
	}

	for _, rule := range rules {
		if rule.DataSourceURL == "" {
			continue
		}
		ds := models.DataSource{
			ID:       uuid.New(),
			Type:     rule.DataSourceType,
			Endpoint: rule.DataSourceURL,
		}
		firingList, err := w.evaluator.EvaluateRule(ctx, rule, ds)
		if err != nil {
			log.Printf("AlertNotificationWorker: evaluate rule %s: %v", rule.ID, err)
			continue
		}

		now := time.Now()
		for _, fa := range firingList {
			// Skip if current time is outside effective window or inside exclusion window.
			if !inEffectiveWindow(rule, now) || inExclusionWindow(rule, now) {
				continue
			}
			key := pendingKey{ruleID: rule.ID, fingerprint: fa.Fingerprint}
			seenThisRun[key] = struct{}{}

			w.pendingMu.Lock()
			state, exists := w.pending[key]
			if !exists {
				state = pendingState{firstSeenAt: time.Now(), notified: false}
				w.pending[key] = state
			}
			w.pendingMu.Unlock()

			// Only fire and notify after condition has held for rule.ForDuration seconds.
			held := time.Since(state.firstSeenAt)
			if held < time.Duration(rule.ForDuration)*time.Second {
				continue
			}
			if state.notified {
				continue
			}

			// Mark as notified so we do not send again until this firing period ends.
			w.pendingMu.Lock()
			w.pending[key] = pendingState{firstSeenAt: state.firstSeenAt, notified: true}
			w.pendingMu.Unlock()

			labelsJSON := "{}"
			if len(fa.Labels) > 0 {
				b, _ := json.Marshal(fa.Labels)
				labelsJSON = string(b)
			}
			annotationsJSON := "{}"
			if len(fa.Annotations) > 0 {
				b, _ := json.Marshal(fa.Annotations)
				annotationsJSON = string(b)
			}

			history := &models.AlertHistory{
				RuleID:      rule.ID,
				Fingerprint: fa.Fingerprint,
				Severity:    rule.Severity,
				Status:      "firing",
				StartedAt:   fa.StartsAt,
				Labels:      labelsJSON,
				Annotations: annotationsJSON,
			}
			if err := w.historyRepo.Create(ctx, history); err != nil {
				log.Printf("AlertNotificationWorker: create alert_history: %v", err)
				continue
			}

			// Create SLA record for this alert if config exists.
			if w.slaSvc != nil {
				if err := w.slaSvc.CreateAlertSLA(ctx, history.ID, rule.ID, rule.Severity, history.StartedAt); err != nil {
					log.Printf("AlertNotificationWorker: create alert_sla: %v", err)
				}
			}

			var renderedContent string
			if rule.TemplateID != nil && w.templateSvc != nil {
				data := map[string]interface{}{
					"ruleName":          rule.Name,
					"severity":          rule.Severity,
					"status":            "firing",
					"startTime":         fa.StartsAt.Format("2006-01-02 15:04:05"),
					"duration":          "0",
					"labels":            labelsJSON,
					"annotations":       annotationsJSON,
					"labelsFormatted":   formatMapToKeyValueLines(labelsJSON),
					"annotationsFormatted": formatMapToKeyValueLines(annotationsJSON),
				}
				if r, err := w.templateSvc.Render(ctx, *rule.TemplateID, data); err == nil {
					renderedContent = r
				} else {
					log.Printf("AlertNotificationWorker: render template %s: %v", rule.TemplateID, err)
				}
			}
			payload := &AlertPayload{
				AlertNo:         history.AlertNo,
				RuleID:          rule.ID,
				RuleName:        rule.Name,
				Severity:        rule.Severity,
				Status:          "firing",
				Description:     rule.Description,
				Labels:         labelsJSON,
				StartedAt:       fa.StartsAt,
				RenderedContent: renderedContent,
			}
			if err := w.sender.SendToRuleChannels(ctx, rule.ID, payload); err != nil {
				log.Printf("AlertNotificationWorker: send to channels for rule %s: %v", rule.ID, err)
			}
			if w.broadcaster != nil {
				w.broadcaster.SendAlertNotification(&AlertNotification{
					AlertID:   history.ID.String(),
					RuleID:    rule.ID.String(),
					RuleName:  rule.Name,
					Severity:  rule.Severity,
					Status:    "firing",
					Labels:    fa.Labels,
					Timestamp: time.Now(),
				})
			}
		}
	}

	// Detect recovery: keys that were notified (firing) but are no longer in seenThisRun.
	now := time.Now()
	w.pendingMu.Lock()
	var recovered []pendingKey
	for key, state := range w.pending {
		if _, seen := seenThisRun[key]; !seen && state.notified {
			recovered = append(recovered, key)
		}
	}
	w.pendingMu.Unlock()

	for _, key := range recovered {
		rule, ok := ruleByID[key.ruleID]
		if !ok {
			continue
		}
		hist, err := w.historyRepo.GetLatestFiringByRuleAndFingerprint(ctx, key.ruleID, key.fingerprint)
		if err != nil || hist == nil {
			log.Printf("AlertNotificationWorker: get latest firing for recovery %s/%s: %v", key.ruleID, key.fingerprint, err)
			continue
		}
		if err := w.historyRepo.MarkResolvedByRuleAndFingerprint(ctx, key.ruleID, key.fingerprint, now); err != nil {
			log.Printf("AlertNotificationWorker: mark resolved %s/%s: %v", key.ruleID, key.fingerprint, err)
			continue
		}
		if w.slaSvc != nil {
			if err := w.slaSvc.MarkResolved(ctx, hist.ID, now); err != nil {
				log.Printf("AlertNotificationWorker: mark alert_sla resolved %s: %v", hist.ID, err)
			}
		}
		dur := now.Sub(hist.StartedAt).Round(time.Second)
		var renderedContent string
		if rule.TemplateID != nil && w.templateSvc != nil {
			data := map[string]interface{}{
				"ruleName":            rule.Name,
				"severity":            rule.Severity,
				"status":              "resolved",
				"startTime":           hist.StartedAt.Format("2006-01-02 15:04:05"),
				"duration":            dur.String(),
				"endTime":             now.Format("2006-01-02 15:04:05"),
				"labels":              hist.Labels,
				"annotations":         hist.Annotations,
				"labelsFormatted":     formatMapToKeyValueLines(hist.Labels),
				"annotationsFormatted": formatMapToKeyValueLines(hist.Annotations),
			}
			if r, err := w.templateSvc.Render(ctx, *rule.TemplateID, data); err == nil {
				renderedContent = r
			} else {
				log.Printf("AlertNotificationWorker: render template for recovery %s: %v", rule.TemplateID, err)
			}
		}
		payload := &AlertPayload{
			AlertNo:         hist.AlertNo,
			RuleID:          rule.ID,
			RuleName:        rule.Name,
			Severity:        rule.Severity,
			Status:          "resolved",
			Description:     rule.Description,
			Labels:          hist.Labels,
			StartedAt:       hist.StartedAt,
			EndedAt:         &now,
			RenderedContent: renderedContent,
		}
		if err := w.sender.SendToRuleChannels(ctx, rule.ID, payload); err != nil {
			log.Printf("AlertNotificationWorker: send recovery to channels for rule %s: %v", rule.ID, err)
		}
		if w.broadcaster != nil {
			w.broadcaster.SendAlertNotification(&AlertNotification{
				AlertID:   hist.ID.String(),
				RuleID:    rule.ID.String(),
				RuleName:  rule.Name,
				Severity:  rule.Severity,
				Status:    "resolved",
				Labels:    nil,
				Timestamp: time.Now(),
			})
		}
	}

	// Remove from pending any (ruleID, fingerprint) that is no longer firing this run (resolved).
	w.pendingMu.Lock()
	for key := range w.pending {
		if _, seen := seenThisRun[key]; !seen {
			delete(w.pending, key)
		}
	}
	w.pendingMu.Unlock()

	return nil
}
