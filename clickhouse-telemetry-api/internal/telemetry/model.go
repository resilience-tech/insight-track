package telemetry

import (
	"context"
	"time"
)

type MetricInput struct {
	Timestamp  time.Time         `json:"timestamp"`
	Name       string            `json:"name"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Metric struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	ServiceID  string            `json:"service_id"`
	Timestamp  time.Time         `json:"timestamp"`
	Name       string            `json:"name"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
	Attributes map[string]string `json:"attributes"`
	IngestedAt time.Time         `json:"ingested_at"`
}

type LogInput struct {
	Timestamp  time.Time         `json:"timestamp"`
	Severity   string            `json:"severity,omitempty"`
	Message    string            `json:"message"`
	TraceID    string            `json:"trace_id,omitempty"`
	SpanID     string            `json:"span_id,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Log struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	ServiceID  string            `json:"service_id"`
	Timestamp  time.Time         `json:"timestamp"`
	Severity   string            `json:"severity"`
	Message    string            `json:"message"`
	TraceID    string            `json:"trace_id,omitempty"`
	SpanID     string            `json:"span_id,omitempty"`
	Attributes map[string]string `json:"attributes"`
	IngestedAt time.Time         `json:"ingested_at"`
}

type SpanInput struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	StartTime    time.Time         `json:"start_time"`
	DurationMS   float64           `json:"duration_ms"`
	Status       string            `json:"status,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

type Span struct {
	ID           string            `json:"id"`
	ProjectID    string            `json:"project_id"`
	ServiceID    string            `json:"service_id"`
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	StartTime    time.Time         `json:"start_time"`
	DurationMS   float64           `json:"duration_ms"`
	Status       string            `json:"status"`
	Attributes   map[string]string `json:"attributes"`
	IngestedAt   time.Time         `json:"ingested_at"`
}

type Summary struct {
	MetricPoints      uint64  `json:"metric_points"`
	LogEntries        uint64  `json:"log_entries"`
	Spans             uint64  `json:"spans"`
	ErrorLogs         uint64  `json:"error_logs"`
	ErrorSpans        uint64  `json:"error_spans"`
	AverageDurationMS float64 `json:"average_span_duration_ms"`
}

type ListFilter struct {
	ProjectID string
	ServiceID string
	From      time.Time
	To        time.Time
	Before    *Cursor
	Limit     int
}

type Cursor struct {
	Time time.Time
	ID   string
}

type MetricFilter struct {
	ListFilter
	Name string
}

type LogFilter struct {
	ListFilter
	Severity string
	Search   string
}

type SpanFilter struct {
	ListFilter
	Status  string
	TraceID string
}

type SummaryFilter struct {
	ProjectID string
	ServiceID string
	From      time.Time
	To        time.Time
}

type Store interface {
	Ping(context.Context) error
	InsertMetrics(context.Context, string, string, []Metric) error
	InsertLogs(context.Context, string, string, []Log) error
	InsertSpans(context.Context, string, string, []Span) error
	ListMetrics(context.Context, MetricFilter) ([]Metric, error)
	ListLogs(context.Context, LogFilter) ([]Log, error)
	ListSpans(context.Context, SpanFilter) ([]Span, error)
	GetSummary(context.Context, SummaryFilter) (Summary, error)
}
