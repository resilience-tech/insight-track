package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

const (
	testToken     = "0123456789abcdef0123456789abcdef"
	testProjectID = "019c3d56-7890-7abc-8def-0123456789ab"
	testServiceID = "019c3d56-7890-7abc-8def-0123456789ac"
)

type fakeStore struct {
	insertedMetrics []telemetry.Metric
	metricResults   []telemetry.Metric
	metricFilter    telemetry.MetricFilter
	pingError       error
}

func (s *fakeStore) Ping(context.Context) error {
	return s.pingError
}

func (s *fakeStore) InsertMetrics(_ context.Context, _, _ string, items []telemetry.Metric) error {
	s.insertedMetrics = append(s.insertedMetrics, items...)
	return nil
}

func (*fakeStore) InsertLogs(context.Context, string, string, []telemetry.Log) error {
	return nil
}

func (*fakeStore) InsertSpans(context.Context, string, string, []telemetry.Span) error {
	return nil
}

func (s *fakeStore) ListMetrics(_ context.Context, filter telemetry.MetricFilter) ([]telemetry.Metric, error) {
	s.metricFilter = filter
	return s.metricResults, nil
}

func (*fakeStore) ListLogs(context.Context, telemetry.LogFilter) ([]telemetry.Log, error) {
	return []telemetry.Log{}, nil
}

func (*fakeStore) ListSpans(context.Context, telemetry.SpanFilter) ([]telemetry.Span, error) {
	return []telemetry.Span{}, nil
}

func (*fakeStore) GetSummary(context.Context, telemetry.SummaryFilter) (telemetry.Summary, error) {
	return telemetry.Summary{}, nil
}

func testAPI(store telemetry.Store) *API {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(store, testToken, logger)
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	t.Parallel()
	handler := testAPI(&fakeStore{}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/v1/projects/"+testProjectID+"/services/"+testServiceID+"/telemetry/metrics", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if response.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("WWW-Authenticate header is missing")
	}
}

func TestIngestMetrics(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	api := testAPI(store)
	api.now = func() time.Time {
		return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	}
	handler := api.Handler()
	body := `{"items":[{"timestamp":"2026-09-04T11:59:00Z","name":"cpu.utilization","value":0.75,"unit":"1","attributes":{"host":"web-1"}}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/projects/"+testProjectID+"/services/"+testServiceID+"/telemetry/metrics", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(store.insertedMetrics) != 1 {
		t.Fatalf("inserted metrics = %d", len(store.insertedMetrics))
	}
	metric := store.insertedMetrics[0]
	if metric.ProjectID != testProjectID || metric.ServiceID != testServiceID || !validUUID(metric.ID) {
		t.Fatalf("metric = %#v", metric)
	}
}

func TestIngestRejectsUnsupportedContentType(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	handler := testAPI(store).Handler()
	request := httptest.NewRequest(http.MethodPost, "/v1/projects/"+testProjectID+"/services/"+testServiceID+"/telemetry/metrics", strings.NewReader(`{"items":[]}`))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnsupportedMediaType)
	}
}

func TestListMetricsRequestsOneExtraRow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{metricResults: []telemetry.Metric{
		{ID: "019c3d56-7890-7abc-8def-0123456789ad", Timestamp: now},
		{ID: "019c3d56-7890-7abc-8def-0123456789ae", Timestamp: now.Add(-time.Second)},
		{ID: "019c3d56-7890-7abc-8def-0123456789af", Timestamp: now.Add(-2 * time.Second)},
	}}
	api := testAPI(store)
	api.now = func() time.Time { return now }
	handler := api.Handler()
	request := httptest.NewRequest(http.MethodGet, "/v1/projects/"+testProjectID+"/services/"+testServiceID+"/telemetry/metrics?limit=2", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if store.metricFilter.Limit != 3 {
		t.Fatalf("store limit = %d, want 3", store.metricFilter.Limit)
	}
	if !strings.Contains(response.Body.String(), `"next_cursor":"`) {
		t.Fatalf("next_cursor missing: %s", response.Body.String())
	}
}

func TestReadinessReportsClickHouseFailure(t *testing.T) {
	t.Parallel()
	store := &fakeStore{pingError: context.DeadlineExceeded}
	handler := testAPI(store).Handler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
