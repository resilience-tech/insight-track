package httpapi

import (
	"net/http"
	"strings"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

func (a *API) getSummary(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	from, to, err := parseWindow(r, a.now())
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	summary, err := a.store.GetSummary(r.Context(), telemetry.SummaryFilter{
		ProjectID: projectID, ServiceID: serviceID, From: from, To: to,
	})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) listMetrics(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	filter, ok := a.listFilter(w, r, projectID, serviceID)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name != "" && !validText(name, 1, maximumNameLength) {
		writeValidation(w, r, "name must contain at most 200 characters.")
		return
	}
	items, err := a.store.ListMetrics(r.Context(), telemetry.MetricFilter{ListFilter: filter, Name: name})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	result, err := buildPage(items, filter.Limit-1, func(item telemetry.Metric) listCursor {
		return listCursor{Time: item.Timestamp, ID: item.ID}
	})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) listLogs(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	filter, ok := a.listFilter(w, r, projectID, serviceID)
	if !ok {
		return
	}
	severity := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("severity")))
	if severity != "" && !validLogSeverity(severity) {
		writeValidation(w, r, "severity must be trace, debug, info, warn, error, or fatal.")
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if !validText(search, 0, maximumSearchLength) {
		writeValidation(w, r, "search must contain at most 200 characters.")
		return
	}
	items, err := a.store.ListLogs(r.Context(), telemetry.LogFilter{
		ListFilter: filter, Severity: severity, Search: search,
	})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	result, err := buildPage(items, filter.Limit-1, func(item telemetry.Log) listCursor {
		return listCursor{Time: item.Timestamp, ID: item.ID}
	})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) listSpans(w http.ResponseWriter, r *http.Request) {
	projectID, serviceID, ok := pathTenant(w, r)
	if !ok {
		return
	}
	filter, ok := a.listFilter(w, r, projectID, serviceID)
	if !ok {
		return
	}
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && status != "unset" && status != "ok" && status != "error" {
		writeValidation(w, r, "status must be unset, ok, or error.")
		return
	}
	traceID := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("trace_id")))
	if !validHexID(traceID, 32, true) {
		writeValidation(w, r, "trace_id must contain 32 hexadecimal characters.")
		return
	}
	items, err := a.store.ListSpans(r.Context(), telemetry.SpanFilter{
		ListFilter: filter, Status: status, TraceID: traceID,
	})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	result, err := buildPage(items, filter.Limit-1, func(item telemetry.Span) listCursor {
		return listCursor{Time: item.StartTime, ID: item.ID}
	})
	if err != nil {
		writeStoreError(a, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) listFilter(w http.ResponseWriter, r *http.Request, projectID, serviceID string) (telemetry.ListFilter, bool) {
	from, to, err := parseWindow(r, a.now())
	if err != nil {
		writeValidation(w, r, err.Error())
		return telemetry.ListFilter{}, false
	}
	limit, cursor, err := parseList(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return telemetry.ListFilter{}, false
	}
	return telemetry.ListFilter{
		ProjectID: projectID, ServiceID: serviceID, From: from, To: to,
		Before: cursor, Limit: limit + 1,
	}, true
}
