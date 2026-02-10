package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"alert-center/internal/models"
)

type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}    `json:"value,omitempty"`
			Values [][]interface{}   `json:"values,omitempty"`
		} `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

type PrometheusClient struct {
	client  *http.Client
	baseURL string
}

func NewPrometheusClient(endpoint string) *PrometheusClient {
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "http://" + endpoint
	}
	return &PrometheusClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: strings.TrimSuffix(endpoint, "/"),
	}
}

func (c *PrometheusClient) Query(ctx context.Context, query string, time string) ([]models.QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if time != "" {
		params.Set("time", time)
	}

	resp, err := c.doRequest(ctx, "/api/v1/query", params)
	if err != nil {
		return nil, err
	}

	return c.parseResults(resp)
}

func (c *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]models.QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start.Format(time.RFC3339Nano))
	params.Set("end", end.Format(time.RFC3339Nano))
	params.Set("step", step)

	resp, err := c.doRequest(ctx, "/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	return c.parseResults(resp)
}

func (c *PrometheusClient) doRequest(ctx context.Context, path string, params url.Values) ([]byte, error) {
	url := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *PrometheusClient) parseResults(data []byte) ([]models.QueryResult, error) {
	var result PrometheusQueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed: %s - %s", result.ErrorType, result.Error)
	}

	results := make([]models.QueryResult, 0)
	for _, r := range result.Data.Result {
		queryResult := models.QueryResult{
			Metric: r.Metric,
		}

		if len(r.Value) >= 2 {
			queryResult.Value = parseValue(r.Value)
			results = append(results, queryResult)
		} else if len(r.Values) > 0 {
			for _, v := range r.Values {
				if len(v) >= 2 {
					queryResult.Values = append(queryResult.Values, models.Sample{
						Timestamp: time.Unix(int64(v[0].(float64)), 0),
						Value:      parseFloat64(v[1]),
					})
				}
			}
			if len(queryResult.Values) > 0 {
				results = append(results, queryResult)
			}
		}
	}

	return results, nil
}

func parseValue(v []interface{}) models.Sample {
	if len(v) < 2 {
		return models.Sample{}
	}
	return models.Sample{
		Timestamp: time.Unix(int64(v[0].(float64)), 0),
		Value:      parseFloat64(v[1]),
	}
}

func parseFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

func (c *PrometheusClient) HealthCheck(ctx context.Context) error {
	params := url.Values{}
	params.Set("query", "up")

	_, err := c.doRequest(ctx, "/api/v1/query", params)
	return err
}

type VictoriaMetricsClient struct {
	prom *PrometheusClient
}

func NewVictoriaMetricsClient(endpoint string) *VictoriaMetricsClient {
	return &VictoriaMetricsClient{
		prom: NewPrometheusClient(endpoint),
	}
}

func (c *VictoriaMetricsClient) Query(ctx context.Context, query string, time string) ([]models.QueryResult, error) {
	return c.prom.Query(ctx, query, time)
}

func (c *VictoriaMetricsClient) QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]models.QueryResult, error) {
	return c.prom.QueryRange(ctx, query, start, end, step)
}

func (c *VictoriaMetricsClient) HealthCheck(ctx context.Context) error {
	return c.prom.HealthCheck(ctx)
}
