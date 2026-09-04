package clickhouse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

const (
	testProjectID = "019c3d56-7890-7abc-8def-0123456789ab"
	testServiceID = "019c3d56-7890-7abc-8def-0123456789ac"
	testEventID   = "019c3d56-7890-7abc-8def-0123456789ad"
)

func TestInsertMetricsUsesJSONEachRowAndAuthentication(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "api" || password != "secret" {
			t.Errorf("BasicAuth = %q, %q, %v", username, password, ok)
		}
		if got := r.URL.Query().Get("database"); got != "insight_track" {
			t.Errorf("database = %q", got)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}
		parts := strings.SplitN(string(raw), "FORMAT JSONEachRow\n", 2)
		if len(parts) != 2 || !strings.Contains(parts[0], "INSERT INTO metrics") {
			t.Errorf("unexpected insert body: %s", raw)
			return
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(parts[1])), &row); err != nil {
			t.Errorf("decode row: %v", err)
			return
		}
		if row["project_id"] != testProjectID || row["service_id"] != testServiceID {
			t.Errorf("tenant row = %#v", row)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, "insight_track", "api", "secret", time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	err = client.InsertMetrics(context.Background(), testProjectID, testServiceID, []telemetry.Metric{{
		ID: testEventID, Timestamp: now, Name: "http.requests", Value: 7,
		Attributes: map[string]string{"method": "GET"}, IngestedAt: now,
	}})
	if err != nil {
		t.Fatalf("InsertMetrics: %v", err)
	}
}

func TestListMetricsUsesBoundParametersAndDecodesRows(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("param_project_id"); got != testProjectID {
			t.Errorf("param_project_id = %q", got)
		}
		if got := r.URL.Query().Get("param_metric_name"); got != "cpu.utilization" {
			t.Errorf("param_metric_name = %q", got)
		}
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), "{project_id:UUID}") || strings.Contains(string(raw), testProjectID) {
			t.Errorf("query is not parameterized: %s", raw)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"id":"019c3d56-7890-7abc-8def-0123456789ad","project_id":"019c3d56-7890-7abc-8def-0123456789ab","service_id":"019c3d56-7890-7abc-8def-0123456789ac","timestamp_ms":1788523200000,"name":"cpu.utilization","value":0.75,"unit":"1","attributes":{"host":"web-1"},"ingested_at_ms":1788523201000}`+"\n")
	}))
	defer server.Close()

	client, err := New(server.URL, "insight_track", "api", "secret", time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	items, err := client.ListMetrics(context.Background(), telemetry.MetricFilter{
		ListFilter: telemetry.ListFilter{
			ProjectID: testProjectID,
			ServiceID: testServiceID,
			From:      time.UnixMilli(1788520000000).UTC(),
			To:        time.UnixMilli(1788530000000).UTC(),
			Limit:     10,
		},
		Name: "cpu.utilization",
	})
	if err != nil {
		t.Fatalf("ListMetrics: %v", err)
	}
	if len(items) != 1 || items[0].Name != "cpu.utilization" || items[0].Attributes["host"] != "web-1" {
		t.Fatalf("items = %#v", items)
	}
}

func TestPingReturnsClickHouseError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := New(server.URL, "insight_track", "api", "secret", time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = client.Ping(context.Background())
	if err == nil || !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("Ping error = %v", err)
	}
}
