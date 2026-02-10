package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"alert-center/internal/models"

	"github.com/google/uuid"
)

type AlertEvaluator struct {
	promClients   map[string]*PrometheusClient
	vmClients     map[string]*VictoriaMetricsClient
	mu            sync.RWMutex
	checkInterval time.Duration
}

func NewAlertEvaluator(checkInterval time.Duration) *AlertEvaluator {
	return &AlertEvaluator{
		promClients:   make(map[string]*PrometheusClient),
		vmClients:     make(map[string]*VictoriaMetricsClient),
		checkInterval: checkInterval,
	}
}

func (e *AlertEvaluator) RegisterDataSource(ds models.DataSource) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch ds.Type {
	case "prometheus":
		e.promClients[ds.ID.String()] = NewPrometheusClient(ds.Endpoint)
	case "victoria-metrics":
		e.vmClients[ds.ID.String()] = NewVictoriaMetricsClient(ds.Endpoint)
	}
}

func (e *AlertEvaluator) UnregisterDataSource(id uuid.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.promClients, id.String())
	delete(e.vmClients, id.String())
}

func (e *AlertEvaluator) GetClient(dsType string, endpoint string) interface{} {
	switch dsType {
	case "prometheus":
		return NewPrometheusClient(endpoint)
	case "victoria-metrics":
		return NewVictoriaMetricsClient(endpoint)
	}
	return nil
}

func (e *AlertEvaluator) parseAnnotations(annotations string) map[string]string {
	var result map[string]string
	if annotations != "" {
		json.Unmarshal([]byte(annotations), &result)
	}
	return result
}

func (e *AlertEvaluator) EvaluateRule(ctx context.Context, rule models.AlertRule, ds models.DataSource) ([]models.FiringAlert, error) {
	var firing []models.FiringAlert

	var client *PrometheusClient
	e.mu.RLock()
	client = e.promClients[ds.ID.String()]
	e.mu.RUnlock()

	if client == nil {
		client = NewPrometheusClient(ds.Endpoint)
	}

	results, err := client.Query(ctx, rule.Expression, "")
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		if e.checkThreshold(result.Value.Value, rule) {
			labels := e.mergeLabels(rule.Labels, result.Metric)
			annotations := e.parseAnnotations(rule.Annotations)
			firing = append(firing, models.FiringAlert{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				Fingerprint: models.GenerateFingerprint(labels),
				Labels:      labels,
				Annotations: annotations,
				StartsAt:    time.Now(),
				Value:       result.Value.Value,
				Status:      "firing",
			})
		}
	}

	return firing, nil
}

func (e *AlertEvaluator) checkThreshold(value float64, rule models.AlertRule) bool {
	return value > 0
}

func (e *AlertEvaluator) mergeLabels(ruleLabels string, metricLabels map[string]string) map[string]string {
	result := make(map[string]string)

	var ruleL map[string]string
	if ruleLabels != "" {
		json.Unmarshal([]byte(ruleLabels), &ruleL)
	}
	if ruleL != nil {
		for k, v := range ruleL {
			result[k] = v
		}
	}

	for k, v := range metricLabels {
		result[k] = v
	}

	return result
}

func (e *AlertEvaluator) EvaluateAllRules(ctx context.Context, rules []models.AlertRule, ds models.DataSource) ([]models.FiringAlert, error) {
	var allFiring []models.FiringAlert

	for _, rule := range rules {
		if rule.Status != 1 {
			continue
		}

		firing, err := e.EvaluateRule(ctx, rule, ds)
		if err != nil {
			log.Printf("Error evaluating rule %s: %v", rule.ID, err)
			continue
		}

		allFiring = append(allFiring, firing...)
	}

	return allFiring, nil
}
