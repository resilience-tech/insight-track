package httpapi

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

type metricBatch struct {
	Items []telemetry.MetricInput `json:"items"`
}

type logBatch struct {
	Items []telemetry.LogInput `json:"items"`
}

type spanBatch struct {
	Items []telemetry.SpanInput `json:"items"`
}

type ingestResponse struct {
	Accepted int `json:"accepted"`
}

func (a *API) ingestMetrics(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	var body metricBatch
	if err := decodeJSON(w, r, &body); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	if len(body.Items) < 1 || len(body.Items) > maximumBatchItems {
		writeValidation(w, r, fmt.Sprintf("items must contain between 1 and %d metrics.", maximumBatchItems))
		return
	}
	now := a.now()
	items := make([]telemetry.Metric, 0, len(body.Items))
	for index, input := range body.Items {
		input.Name = strings.TrimSpace(input.Name)
		input.Unit = strings.TrimSpace(input.Unit)
		if !validTelemetryTime(input.Timestamp, now) {
			writeValidation(w, r, fmt.Sprintf("items[%d].timestamp is required and must not be in the future.", index))
			return
		}
		if !validText(input.Name, 1, maximumNameLength) {
			writeValidation(w, r, fmt.Sprintf("items[%d].name must contain 1-%d characters.", index, maximumNameLength))
			return
		}
		if !validText(input.Unit, 0, maximumUnitLength) {
			writeValidation(w, r, fmt.Sprintf("items[%d].unit must contain at most %d characters.", index, maximumUnitLength))
			return
		}
		if math.IsNaN(input.Value) || math.IsInf(input.Value, 0) {
			writeValidation(w, r, fmt.Sprintf("items[%d].value must be a finite number.", index))
			return
		}
		if !validateAttributes(input.Attributes) {
			writeValidation(w, r, fmt.Sprintf("items[%d].attributes exceed the supported size limits.", index))
			return
		}
		id, err := newUUID()
		if err != nil {
			writeStoreError(a, w, r, err)
			return
		}
		items = append(items, telemetry.Metric{
			ID: id, ProjectID: projectID, ServiceID: serviceID,
			Timestamp: input.Timestamp.UTC(), Name: input.Name, Value: input.Value,
			Unit: input.Unit, Attributes: input.Attributes, IngestedAt: now,
		})
	}
	if err := a.store.InsertMetrics(r.Context(), projectID, serviceID, items); err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ingestResponse{Accepted: len(items)})
}

func (a *API) ingestLogs(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	var body logBatch
	if err := decodeJSON(w, r, &body); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	if len(body.Items) < 1 || len(body.Items) > maximumBatchItems {
		writeValidation(w, r, fmt.Sprintf("items must contain between 1 and %d logs.", maximumBatchItems))
		return
	}
	now := a.now()
	items := make([]telemetry.Log, 0, len(body.Items))
	for index, input := range body.Items {
		input.Severity = strings.ToLower(strings.TrimSpace(input.Severity))
		if input.Severity == "" {
			input.Severity = "info"
		}
		input.TraceID = strings.ToLower(strings.TrimSpace(input.TraceID))
		input.SpanID = strings.ToLower(strings.TrimSpace(input.SpanID))
		if !validTelemetryTime(input.Timestamp, now) {
			writeValidation(w, r, fmt.Sprintf("items[%d].timestamp is required and must not be in the future.", index))
			return
		}
		if !validLogSeverity(input.Severity) {
			writeValidation(w, r, fmt.Sprintf("items[%d].severity is invalid.", index))
			return
		}
		if !validText(input.Message, 1, maximumLogMessage) {
			writeValidation(w, r, fmt.Sprintf("items[%d].message must contain 1-%d characters.", index, maximumLogMessage))
			return
		}
		if !validHexID(input.TraceID, 32, true) || !validHexID(input.SpanID, 16, true) {
			writeValidation(w, r, fmt.Sprintf("items[%d] contains an invalid trace_id or span_id.", index))
			return
		}
		if input.SpanID != "" && input.TraceID == "" {
			writeValidation(w, r, fmt.Sprintf("items[%d].trace_id is required when span_id is present.", index))
			return
		}
		if !validateAttributes(input.Attributes) {
			writeValidation(w, r, fmt.Sprintf("items[%d].attributes exceed the supported size limits.", index))
			return
		}
		id, err := newUUID()
		if err != nil {
			writeStoreError(a, w, r, err)
			return
		}
		items = append(items, telemetry.Log{
			ID: id, ProjectID: projectID, ServiceID: serviceID,
			Timestamp: input.Timestamp.UTC(), Severity: input.Severity, Message: input.Message,
			TraceID: input.TraceID, SpanID: input.SpanID, Attributes: input.Attributes, IngestedAt: now,
		})
	}
	if err := a.store.InsertLogs(r.Context(), projectID, serviceID, items); err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ingestResponse{Accepted: len(items)})
}

func (a *API) ingestSpans(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	var body spanBatch
	if err := decodeJSON(w, r, &body); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	if len(body.Items) < 1 || len(body.Items) > maximumBatchItems {
		writeValidation(w, r, fmt.Sprintf("items must contain between 1 and %d spans.", maximumBatchItems))
		return
	}
	now := a.now()
	items := make([]telemetry.Span, 0, len(body.Items))
	for index, input := range body.Items {
		input.TraceID = strings.ToLower(strings.TrimSpace(input.TraceID))
		input.SpanID = strings.ToLower(strings.TrimSpace(input.SpanID))
		input.ParentSpanID = strings.ToLower(strings.TrimSpace(input.ParentSpanID))
		input.Name = strings.TrimSpace(input.Name)
		input.Status = strings.ToLower(strings.TrimSpace(input.Status))
		if input.Status == "" {
			input.Status = "unset"
		}
		if !validTelemetryTime(input.StartTime, now) {
			writeValidation(w, r, fmt.Sprintf("items[%d].start_time is required and must not be in the future.", index))
			return
		}
		if !validHexID(input.TraceID, 32, false) || !validHexID(input.SpanID, 16, false) || !validHexID(input.ParentSpanID, 16, true) {
			writeValidation(w, r, fmt.Sprintf("items[%d] contains an invalid trace or span identifier.", index))
			return
		}
		if !validText(input.Name, 1, maximumNameLength) {
			writeValidation(w, r, fmt.Sprintf("items[%d].name must contain 1-%d characters.", index, maximumNameLength))
			return
		}
		if input.DurationMS < 0 || input.DurationMS > maximumSpanDurationMS || math.IsNaN(input.DurationMS) || math.IsInf(input.DurationMS, 0) {
			writeValidation(w, r, fmt.Sprintf("items[%d].duration_ms is outside the supported range.", index))
			return
		}
		if input.Status != "unset" && input.Status != "ok" && input.Status != "error" {
			writeValidation(w, r, fmt.Sprintf("items[%d].status must be unset, ok, or error.", index))
			return
		}
		if !validateAttributes(input.Attributes) {
			writeValidation(w, r, fmt.Sprintf("items[%d].attributes exceed the supported size limits.", index))
			return
		}
		id, err := newUUID()
		if err != nil {
			writeStoreError(a, w, r, err)
			return
		}
		items = append(items, telemetry.Span{
			ID: id, ProjectID: projectID, ServiceID: serviceID,
			TraceID: input.TraceID, SpanID: input.SpanID, ParentSpanID: input.ParentSpanID,
			Name: input.Name, StartTime: input.StartTime.UTC(), DurationMS: input.DurationMS,
			Status: input.Status, Attributes: input.Attributes, IngestedAt: now,
		})
	}
	if err := a.store.InsertSpans(r.Context(), projectID, serviceID, items); err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ingestResponse{Accepted: len(items)})
}

func validLogSeverity(value string) bool {
	switch value {
	case "trace", "debug", "info", "warn", "error", "fatal":
		return true
	default:
		return false
	}
}
