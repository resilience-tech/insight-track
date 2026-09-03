package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/example/project-tenancy/internal/store"
)

const maxBodyBytes = 1 << 20

type optionalString struct {
	Set   bool
	Value *string
}

type optionalJSON struct {
	Set   bool
	Value json.RawMessage
}

func (value *optionalJSON) UnmarshalJSON(raw []byte) error {
	value.Set = true
	value.Value = append(value.Value[:0], raw...)
	return nil
}

func (value *optionalString) UnmarshalJSON(raw []byte) error {
	value.Set = true
	if string(raw) == "null" {
		value.Value = nil
		return nil
	}
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return errors.New("must be a string or null")
	}
	value.Value = &decoded
	return nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
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

func writeProblem(w http.ResponseWriter, status int, slug, title, detail, requestID string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	body := map[string]any{
		"type":   "https://project-tenancy.example/problems/" + slug,
		"title":  title,
		"status": status,
	}
	if detail != "" {
		body["detail"] = detail
	}
	if requestID != "" {
		body["request_id"] = requestID
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (a *API) writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := requestIDFromContext(r.Context())
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeProblem(w, http.StatusNotFound, "not-found", "Resource not found", "The resource does not exist or is not visible to this user.", requestID)
	case errors.Is(err, store.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "forbidden", "Operation not permitted", "The current user cannot perform this operation.", requestID)
	case errors.Is(err, store.ErrConflict):
		writeProblem(w, http.StatusConflict, "conflict", "Resource state conflict", "The requested operation conflicts with the current resource state.", requestID)
	case errors.Is(err, store.ErrPrecondition):
		writeProblem(w, http.StatusPreconditionFailed, "precondition-failed", "Resource version changed", "If-Match does not match the current resource version.", requestID)
	case errors.Is(err, store.ErrValidation):
		writeProblem(w, http.StatusUnprocessableEntity, "validation-error", "Request validation failed", "One or more request values are invalid.", requestID)
	default:
		a.logger.Error("request failed", "request_id", requestID, "error", err)
		writeProblem(w, http.StatusInternalServerError, "internal-error", "Internal server error", "An unexpected error occurred.", requestID)
	}
}

func parsePage(r *http.Request) (int, string, error) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, "", errors.New("limit must be between 1 and 100")
		}
		limit = parsed
	}
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if len(cursor) > 2000 {
		return 0, "", errors.New("cursor must contain at most 2000 characters")
	}
	return limit, cursor, nil
}

func parseIfMatch(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if len(raw) < 3 || raw[0] != '"' || raw[len(raw)-1] != '"' || strings.Contains(raw[1:len(raw)-1], `"`) {
		return 0, errors.New(`If-Match must contain one strong version ETag, for example "3"`)
	}
	version, err := strconv.ParseInt(raw[1:len(raw)-1], 10, 64)
	if err != nil || version < 1 {
		return 0, errors.New("If-Match contains an invalid version")
	}
	return version, nil
}

func setVersionHeaders(w http.ResponseWriter, version int64) {
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, version))
}

func validateUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	compact := strings.ReplaceAll(value, "-", "")
	_, err := hex.DecodeString(compact)
	return err == nil
}

func validateText(value string, min, max int) bool {
	count := utf8.RuneCountInString(strings.TrimSpace(value))
	return count >= min && count <= max
}

func validateNullableText(value *string, max int) bool {
	return value == nil || utf8.RuneCountInString(*value) <= max
}

func validateEmail(value string) bool {
	if len(value) > 320 || strings.TrimSpace(value) != value {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value) && strings.Contains(value, "@")
}

func validateURL(value *string) bool {
	if value == nil {
		return true
	}
	if *value == "" {
		return false
	}
	parsed, err := url.ParseRequestURI(*value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return scheme == "https" || scheme == "http"
}

func validateJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}

func newRequestID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
