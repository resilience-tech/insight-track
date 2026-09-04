package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

const maximumErrorBodyBytes = 64 << 10

type Client struct {
	baseURL  *url.URL
	database string
	username string
	password string
	http     *http.Client
}

func New(rawURL, database, username, password string, timeout time.Duration) (*Client, error) {
	baseURL, err := url.Parse(rawURL)
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, errors.New("ClickHouse URL must be an absolute HTTP(S) URL")
	}
	baseURL.RawQuery = ""
	baseURL.Fragment = ""
	return &Client{
		baseURL:  baseURL,
		database: database,
		username: username,
		password: password,
		http:     &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	response, err := c.post(ctx, strings.NewReader("SELECT 1"), nil)
	if err != nil {
		return fmt.Errorf("ping ClickHouse: %w", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("read ClickHouse ping: %w", err)
	}
	return nil
}

func (c *Client) InsertMetrics(ctx context.Context, projectID, serviceID string, items []telemetry.Metric) error {
	type row struct {
		ID         string            `json:"id"`
		ProjectID  string            `json:"project_id"`
		ServiceID  string            `json:"service_id"`
		Timestamp  string            `json:"timestamp"`
		Name       string            `json:"name"`
		Value      float64           `json:"value"`
		Unit       string            `json:"unit"`
		Attributes map[string]string `json:"attributes"`
		IngestedAt string            `json:"ingested_at"`
	}
	rows := make([]row, 0, len(items))
	for _, item := range items {
		rows = append(rows, row{
			ID: item.ID, ProjectID: projectID, ServiceID: serviceID,
			Timestamp: clickHouseTime(item.Timestamp), Name: item.Name, Value: item.Value,
			Unit: item.Unit, Attributes: nonNilAttributes(item.Attributes),
			IngestedAt: clickHouseTime(item.IngestedAt),
		})
	}
	return insertRows(ctx, c, `INSERT INTO metrics
		(id, project_id, service_id, timestamp, name, value, unit, attributes, ingested_at)
		FORMAT JSONEachRow`, rows)
}

func (c *Client) InsertLogs(ctx context.Context, projectID, serviceID string, items []telemetry.Log) error {
	type row struct {
		ID         string            `json:"id"`
		ProjectID  string            `json:"project_id"`
		ServiceID  string            `json:"service_id"`
		Timestamp  string            `json:"timestamp"`
		Severity   string            `json:"severity"`
		Message    string            `json:"message"`
		TraceID    string            `json:"trace_id"`
		SpanID     string            `json:"span_id"`
		Attributes map[string]string `json:"attributes"`
		IngestedAt string            `json:"ingested_at"`
	}
	rows := make([]row, 0, len(items))
	for _, item := range items {
		rows = append(rows, row{
			ID: item.ID, ProjectID: projectID, ServiceID: serviceID,
			Timestamp: clickHouseTime(item.Timestamp), Severity: item.Severity, Message: item.Message,
			TraceID: item.TraceID, SpanID: item.SpanID, Attributes: nonNilAttributes(item.Attributes),
			IngestedAt: clickHouseTime(item.IngestedAt),
		})
	}
	return insertRows(ctx, c, `INSERT INTO logs
		(id, project_id, service_id, timestamp, severity, message, trace_id, span_id, attributes, ingested_at)
		FORMAT JSONEachRow`, rows)
}

func (c *Client) InsertSpans(ctx context.Context, projectID, serviceID string, items []telemetry.Span) error {
	type row struct {
		ID           string            `json:"id"`
		ProjectID    string            `json:"project_id"`
		ServiceID    string            `json:"service_id"`
		TraceID      string            `json:"trace_id"`
		SpanID       string            `json:"span_id"`
		ParentSpanID string            `json:"parent_span_id"`
		Name         string            `json:"name"`
		StartTime    string            `json:"start_time"`
		DurationMS   float64           `json:"duration_ms"`
		Status       string            `json:"status"`
		Attributes   map[string]string `json:"attributes"`
		IngestedAt   string            `json:"ingested_at"`
	}
	rows := make([]row, 0, len(items))
	for _, item := range items {
		rows = append(rows, row{
			ID: item.ID, ProjectID: projectID, ServiceID: serviceID,
			TraceID: item.TraceID, SpanID: item.SpanID, ParentSpanID: item.ParentSpanID,
			Name: item.Name, StartTime: clickHouseTime(item.StartTime), DurationMS: item.DurationMS,
			Status: item.Status, Attributes: nonNilAttributes(item.Attributes),
			IngestedAt: clickHouseTime(item.IngestedAt),
		})
	}
	return insertRows(ctx, c, `INSERT INTO spans
		(id, project_id, service_id, trace_id, span_id, parent_span_id, name, start_time,
		 duration_ms, status, attributes, ingested_at)
		FORMAT JSONEachRow`, rows)
}

func (c *Client) ListMetrics(ctx context.Context, filter telemetry.MetricFilter) ([]telemetry.Metric, error) {
	where, params := listWhere("m", "timestamp", filter.ListFilter)
	if filter.Name != "" {
		where = append(where, "m.name = {metric_name:String}")
		params.Set("param_metric_name", filter.Name)
	}
	query := fmt.Sprintf(`SELECT
		toString(m.id) AS id,
		toString(m.project_id) AS project_id,
		toString(m.service_id) AS service_id,
		toUnixTimestamp64Milli(m.timestamp) AS timestamp_ms,
		m.name AS name, m.value AS value, m.unit AS unit, m.attributes AS attributes,
		toUnixTimestamp64Milli(m.ingested_at) AS ingested_at_ms
	FROM metrics AS m
	WHERE %s
	ORDER BY m.timestamp DESC, m.id DESC
	LIMIT %d
	FORMAT JSONEachRow`, strings.Join(where, " AND "), filter.Limit)
	type row struct {
		ID           string            `json:"id"`
		ProjectID    string            `json:"project_id"`
		ServiceID    string            `json:"service_id"`
		TimestampMS  int64             `json:"timestamp_ms"`
		Name         string            `json:"name"`
		Value        float64           `json:"value"`
		Unit         string            `json:"unit"`
		Attributes   map[string]string `json:"attributes"`
		IngestedAtMS int64             `json:"ingested_at_ms"`
	}
	rows, err := queryRows[row](ctx, c, query, params)
	if err != nil {
		return nil, err
	}
	items := make([]telemetry.Metric, 0, len(rows))
	for _, value := range rows {
		items = append(items, telemetry.Metric{
			ID: value.ID, ProjectID: value.ProjectID, ServiceID: value.ServiceID,
			Timestamp: time.UnixMilli(value.TimestampMS).UTC(), Name: value.Name, Value: value.Value,
			Unit: value.Unit, Attributes: nonNilAttributes(value.Attributes),
			IngestedAt: time.UnixMilli(value.IngestedAtMS).UTC(),
		})
	}
	return items, nil
}

func (c *Client) ListLogs(ctx context.Context, filter telemetry.LogFilter) ([]telemetry.Log, error) {
	where, params := listWhere("l", "timestamp", filter.ListFilter)
	if filter.Severity != "" {
		where = append(where, "l.severity = {severity:String}")
		params.Set("param_severity", filter.Severity)
	}
	if filter.Search != "" {
		where = append(where, "positionCaseInsensitiveUTF8(l.message, {search:String}) > 0")
		params.Set("param_search", filter.Search)
	}
	query := fmt.Sprintf(`SELECT
		toString(l.id) AS id,
		toString(l.project_id) AS project_id,
		toString(l.service_id) AS service_id,
		toUnixTimestamp64Milli(l.timestamp) AS timestamp_ms,
		l.severity AS severity, l.message AS message, l.trace_id AS trace_id,
		l.span_id AS span_id, l.attributes AS attributes,
		toUnixTimestamp64Milli(l.ingested_at) AS ingested_at_ms
	FROM logs AS l
	WHERE %s
	ORDER BY l.timestamp DESC, l.id DESC
	LIMIT %d
	FORMAT JSONEachRow`, strings.Join(where, " AND "), filter.Limit)
	type row struct {
		ID           string            `json:"id"`
		ProjectID    string            `json:"project_id"`
		ServiceID    string            `json:"service_id"`
		TimestampMS  int64             `json:"timestamp_ms"`
		Severity     string            `json:"severity"`
		Message      string            `json:"message"`
		TraceID      string            `json:"trace_id"`
		SpanID       string            `json:"span_id"`
		Attributes   map[string]string `json:"attributes"`
		IngestedAtMS int64             `json:"ingested_at_ms"`
	}
	rows, err := queryRows[row](ctx, c, query, params)
	if err != nil {
		return nil, err
	}
	items := make([]telemetry.Log, 0, len(rows))
	for _, value := range rows {
		items = append(items, telemetry.Log{
			ID: value.ID, ProjectID: value.ProjectID, ServiceID: value.ServiceID,
			Timestamp: time.UnixMilli(value.TimestampMS).UTC(), Severity: value.Severity,
			Message: value.Message, TraceID: value.TraceID, SpanID: value.SpanID,
			Attributes: nonNilAttributes(value.Attributes),
			IngestedAt: time.UnixMilli(value.IngestedAtMS).UTC(),
		})
	}
	return items, nil
}

func (c *Client) ListSpans(ctx context.Context, filter telemetry.SpanFilter) ([]telemetry.Span, error) {
	where, params := listWhere("s", "start_time", filter.ListFilter)
	if filter.Status != "" {
		where = append(where, "s.status = {status:String}")
		params.Set("param_status", filter.Status)
	}
	if filter.TraceID != "" {
		where = append(where, "s.trace_id = {trace_id:String}")
		params.Set("param_trace_id", filter.TraceID)
	}
	query := fmt.Sprintf(`SELECT
		toString(s.id) AS id,
		toString(s.project_id) AS project_id,
		toString(s.service_id) AS service_id,
		s.trace_id AS trace_id, s.span_id AS span_id, s.parent_span_id AS parent_span_id,
		s.name AS name,
		toUnixTimestamp64Milli(s.start_time) AS start_time_ms,
		s.duration_ms AS duration_ms, s.status AS status, s.attributes AS attributes,
		toUnixTimestamp64Milli(s.ingested_at) AS ingested_at_ms
	FROM spans AS s
	WHERE %s
	ORDER BY s.start_time DESC, s.id DESC
	LIMIT %d
	FORMAT JSONEachRow`, strings.Join(where, " AND "), filter.Limit)
	type row struct {
		ID           string            `json:"id"`
		ProjectID    string            `json:"project_id"`
		ServiceID    string            `json:"service_id"`
		TraceID      string            `json:"trace_id"`
		SpanID       string            `json:"span_id"`
		ParentSpanID string            `json:"parent_span_id"`
		Name         string            `json:"name"`
		StartTimeMS  int64             `json:"start_time_ms"`
		DurationMS   float64           `json:"duration_ms"`
		Status       string            `json:"status"`
		Attributes   map[string]string `json:"attributes"`
		IngestedAtMS int64             `json:"ingested_at_ms"`
	}
	rows, err := queryRows[row](ctx, c, query, params)
	if err != nil {
		return nil, err
	}
	items := make([]telemetry.Span, 0, len(rows))
	for _, value := range rows {
		items = append(items, telemetry.Span{
			ID: value.ID, ProjectID: value.ProjectID, ServiceID: value.ServiceID,
			TraceID: value.TraceID, SpanID: value.SpanID, ParentSpanID: value.ParentSpanID,
			Name: value.Name, StartTime: time.UnixMilli(value.StartTimeMS).UTC(),
			DurationMS: value.DurationMS, Status: value.Status,
			Attributes: nonNilAttributes(value.Attributes),
			IngestedAt: time.UnixMilli(value.IngestedAtMS).UTC(),
		})
	}
	return items, nil
}

func (c *Client) GetSummary(ctx context.Context, filter telemetry.SummaryFilter) (telemetry.Summary, error) {
	params := url.Values{
		"param_project_id": {filter.ProjectID},
		"param_service_id": {filter.ServiceID},
		"param_from_ms":    {strconv.FormatInt(filter.From.UnixMilli(), 10)},
		"param_to_ms":      {strconv.FormatInt(filter.To.UnixMilli(), 10)},
	}
	metricWhere := summaryWhere("timestamp")
	spanWhere := summaryWhere("start_time")
	query := fmt.Sprintf(`SELECT
		(SELECT count() FROM metrics WHERE %s) AS metric_points,
		(SELECT count() FROM logs WHERE %s) AS log_entries,
		(SELECT count() FROM spans WHERE %s) AS spans,
		(SELECT countIf(severity IN ('error', 'fatal')) FROM logs WHERE %s) AS error_logs,
		(SELECT countIf(status = 'error') FROM spans WHERE %s) AS error_spans,
		(SELECT ifNull(avgOrNull(duration_ms), 0) FROM spans WHERE %s) AS average_span_duration_ms
	FORMAT JSONEachRow`, metricWhere, metricWhere, spanWhere, metricWhere, spanWhere, spanWhere)
	type row struct {
		MetricPoints      uint64  `json:"metric_points"`
		LogEntries        uint64  `json:"log_entries"`
		Spans             uint64  `json:"spans"`
		ErrorLogs         uint64  `json:"error_logs"`
		ErrorSpans        uint64  `json:"error_spans"`
		AverageDurationMS float64 `json:"average_span_duration_ms"`
	}
	rows, err := queryRows[row](ctx, c, query, params)
	if err != nil {
		return telemetry.Summary{}, err
	}
	if len(rows) != 1 {
		return telemetry.Summary{}, errors.New("ClickHouse summary returned an unexpected row count")
	}
	return telemetry.Summary{
		MetricPoints: rows[0].MetricPoints, LogEntries: rows[0].LogEntries, Spans: rows[0].Spans,
		ErrorLogs: rows[0].ErrorLogs, ErrorSpans: rows[0].ErrorSpans,
		AverageDurationMS: rows[0].AverageDurationMS,
	}, nil
}

func insertRows[T any](ctx context.Context, c *Client, statement string, rows []T) error {
	var body bytes.Buffer
	body.WriteString(statement)
	body.WriteByte('\n')
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("encode ClickHouse row: %w", err)
		}
	}
	response, err := c.post(ctx, &body, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("read ClickHouse insert response: %w", err)
	}
	return nil
}

func queryRows[T any](ctx context.Context, c *Client, query string, params url.Values) ([]T, error) {
	response, err := c.post(ctx, strings.NewReader(query), params)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	items := make([]T, 0)
	decoder := json.NewDecoder(response.Body)
	for {
		var item T
		if err := decoder.Decode(&item); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode ClickHouse response: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *Client) post(ctx context.Context, body io.Reader, params url.Values) (*http.Response, error) {
	endpoint := *c.baseURL
	query := endpoint.Query()
	query.Set("database", c.database)
	query.Set("output_format_json_quote_64bit_integers", "0")
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create ClickHouse request: %w", err)
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	request.Header.Set("Accept", "application/x-ndjson")
	request.SetBasicAuth(c.username, c.password)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send ClickHouse request: %w", err)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response, nil
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, maximumErrorBodyBytes))
	detail := strings.TrimSpace(string(raw))
	if detail == "" {
		detail = response.Status
	}
	return nil, fmt.Errorf("ClickHouse returned %d: %s", response.StatusCode, detail)
}

func listWhere(tableAlias, timeColumn string, filter telemetry.ListFilter) ([]string, url.Values) {
	qualifiedTime := tableAlias + "." + timeColumn
	where := []string{
		tableAlias + ".project_id = {project_id:UUID}",
		tableAlias + ".service_id = {service_id:UUID}",
		fmt.Sprintf("%s >= fromUnixTimestamp64Milli({from_ms:Int64})", qualifiedTime),
		fmt.Sprintf("%s < fromUnixTimestamp64Milli({to_ms:Int64})", qualifiedTime),
	}
	params := url.Values{
		"param_project_id": {filter.ProjectID},
		"param_service_id": {filter.ServiceID},
		"param_from_ms":    {strconv.FormatInt(filter.From.UnixMilli(), 10)},
		"param_to_ms":      {strconv.FormatInt(filter.To.UnixMilli(), 10)},
	}
	if filter.Before != nil {
		where = append(where, fmt.Sprintf("(%s, %s.id) < (fromUnixTimestamp64Milli({before_ms:Int64}), {before_id:UUID})", qualifiedTime, tableAlias))
		params.Set("param_before_ms", strconv.FormatInt(filter.Before.Time.UnixMilli(), 10))
		params.Set("param_before_id", filter.Before.ID)
	}
	return where, params
}

func summaryWhere(timeColumn string) string {
	return fmt.Sprintf(`project_id = {project_id:UUID}
		AND service_id = {service_id:UUID}
		AND %s >= fromUnixTimestamp64Milli({from_ms:Int64})
		AND %s < fromUnixTimestamp64Milli({to_ms:Int64})`, timeColumn, timeColumn)
}

func clickHouseTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000")
}

func nonNilAttributes(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}
