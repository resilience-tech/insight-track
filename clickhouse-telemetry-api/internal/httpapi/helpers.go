package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

const (
	maximumBodyBytes      = 2 << 20
	maximumBatchItems     = 1000
	defaultPageLimit      = 100
	maximumPageLimit      = 1000
	maximumQueryRange     = 31 * 24 * time.Hour
	maximumClockSkew      = 5 * time.Minute
	maximumAttributes     = 64
	maximumAttributeKey   = 128
	maximumAttributeValue = 1024
	maximumNameLength     = 200
	maximumUnitLength     = 64
	maximumLogMessage     = 8192
	maximumSearchLength   = 200
	maximumCursorLength   = 2048
	maximumSpanDurationMS = 24 * 60 * 60 * 1000
)

var (
	errBodyTooLarge     = errors.New("request body is too large")
	errUnsupportedMedia = errors.New("Content-Type must be application/json")
)

type page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

type listCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return errUnsupportedMedia
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximumBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maximumError *http.MaxBytesError
		if errors.As(err, &maximumError) {
			return errBodyTooLarge
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, slug, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="clickhouse-telemetry-api"`)
	}
	w.WriteHeader(status)
	body := map[string]any{
		"type":   "https://insight-track.example/problems/" + slug,
		"title":  title,
		"status": status,
	}
	if detail != "" {
		body["detail"] = detail
	}
	if requestID := requestIDFromContext(r.Context()); requestID != "" {
		body["request_id"] = requestID
	}
	_ = json.NewEncoder(w).Encode(body)
}

func writeDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errUnsupportedMedia):
		writeProblem(w, r, http.StatusUnsupportedMediaType, "unsupported-media-type", "Unsupported media type", err.Error())
	case errors.Is(err, errBodyTooLarge):
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload-too-large", "Payload too large", "The request body exceeds 2 MiB.")
	default:
		writeProblem(w, r, http.StatusUnprocessableEntity, "validation-error", "Request validation failed", err.Error())
	}
}

func writeValidation(w http.ResponseWriter, r *http.Request, detail string) {
	writeProblem(w, r, http.StatusUnprocessableEntity, "validation-error", "Request validation failed", detail)
}

func writeStoreError(a *API, w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Error("ClickHouse operation failed", "request_id", requestIDFromContext(r.Context()), "error", err)
	writeProblem(w, r, http.StatusBadGateway, "telemetry-store-error", "Telemetry store error", "The telemetry store could not complete the request.")
}

func pathTenant(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	projectID := r.PathValue("projectId")
	serviceID := r.PathValue("serviceId")
	if !validUUID(projectID) || !validUUID(serviceID) {
		writeValidation(w, r, "projectId and serviceId must be UUIDs.")
		return "", "", false
	}
	return projectID, serviceID, true
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	compact := strings.ReplaceAll(value, "-", "")
	_, err := hex.DecodeString(compact)
	return err == nil
}

func newUUID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}

func parseWindow(r *http.Request, now time.Time) (time.Time, time.Time, error) {
	to := now.UTC()
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("to must be an RFC3339 timestamp")
		}
		to = parsed.UTC()
	}
	from := to.Add(-time.Hour)
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("from must be an RFC3339 timestamp")
		}
		from = parsed.UTC()
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, errors.New("from must be earlier than to")
	}
	if to.Sub(from) > maximumQueryRange {
		return time.Time{}, time.Time{}, errors.New("time range must not exceed 31 days")
	}
	return from, to, nil
}

func parseList(r *http.Request) (int, *telemetry.Cursor, error) {
	limit := defaultPageLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maximumPageLimit {
			return 0, nil, fmt.Errorf("limit must be between 1 and %d", maximumPageLimit)
		}
		limit = parsed
	}
	rawCursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if rawCursor == "" {
		return limit, nil, nil
	}
	if len(rawCursor) > maximumCursorLength {
		return 0, nil, fmt.Errorf("cursor must contain at most %d characters", maximumCursorLength)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(rawCursor)
	if err != nil {
		return 0, nil, errors.New("cursor is invalid")
	}
	var cursor listCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.Time.IsZero() || !validUUID(cursor.ID) {
		return 0, nil, errors.New("cursor is invalid")
	}
	return limit, &telemetry.Cursor{Time: cursor.Time.UTC(), ID: cursor.ID}, nil
}

func buildPage[T any](items []T, limit int, cursorFor func(T) listCursor) (page[T], error) {
	result := page[T]{Items: items}
	if len(items) <= limit {
		return result, nil
	}
	result.Items = items[:limit]
	cursor := cursorFor(result.Items[limit-1])
	raw, err := json.Marshal(cursor)
	if err != nil {
		return page[T]{}, fmt.Errorf("encode cursor: %w", err)
	}
	next := base64.RawURLEncoding.EncodeToString(raw)
	result.NextCursor = &next
	return result, nil
}

func validText(value string, minimum, maximum int) bool {
	length := utf8.RuneCountInString(strings.TrimSpace(value))
	return length >= minimum && length <= maximum
}

func validHexID(value string, size int, optional bool) bool {
	if value == "" {
		return optional
	}
	if len(value) != size {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validateAttributes(attributes map[string]string) bool {
	if len(attributes) > maximumAttributes {
		return false
	}
	for key, value := range attributes {
		if strings.TrimSpace(key) != key || !validText(key, 1, maximumAttributeKey) || utf8.RuneCountInString(value) > maximumAttributeValue {
			return false
		}
	}
	return true
}

func validTelemetryTime(value, now time.Time) bool {
	return !value.IsZero() && !value.After(now.Add(maximumClockSkew))
}
