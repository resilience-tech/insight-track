package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/resilience-tech/insight-track/clickhouse-telemetry-api/internal/telemetry"
)

type contextKey string

const requestIDKey contextKey = "request_id"

type API struct {
	store  telemetry.Store
	token  [sha256.Size]byte
	logger *slog.Logger
	now    func() time.Time
}

func New(store telemetry.Store, token string, logger *slog.Logger) *API {
	if logger == nil {
		logger = slog.Default()
	}
	return &API{
		store:  store,
		token:  sha256.Sum256([]byte(token)),
		logger: logger,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (a *API) Handler() http.Handler {
	protected := http.NewServeMux()
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}/telemetry/summary", a.getSummary)
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}/telemetry/metrics", a.listMetrics)
	protected.HandleFunc("POST /v1/projects/{projectId}/services/{serviceId}/telemetry/metrics", a.ingestMetrics)
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}/telemetry/logs", a.listLogs)
	protected.HandleFunc("POST /v1/projects/{projectId}/services/{serviceId}/telemetry/logs", a.ingestLogs)
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}/telemetry/traces", a.listSpans)
	protected.HandleFunc("POST /v1/projects/{projectId}/services/{serviceId}/telemetry/traces", a.ingestSpans)

	root := http.NewServeMux()
	root.HandleFunc("GET /health/live", a.liveness)
	root.HandleFunc("GET /health/ready", a.readiness)
	root.Handle("/v1/", a.authorize(protected))

	return a.requestID(a.recoverPanic(a.accessLog(a.securityHeaders(root))))
}

func (a *API) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", "A bearer token is required.")
			return
		}
		provided := sha256.Sum256([]byte(parts[1]))
		if subtle.ConstantTimeCompare(provided[:], a.token[:]) != 1 {
			writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", "The bearer token is invalid.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if !validUUID(requestID) {
			var err error
			requestID, err = newUUID()
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
	})
}

func (a *API) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", "request_id", requestIDFromContext(r.Context()), "panic", recovered, "stack", string(debug.Stack()))
				writeProblem(w, r, http.StatusInternalServerError, "internal-error", "Internal server error", "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	written, err := r.ResponseWriter.Write(body)
	r.bytes += written
	return written, err
}

func (a *API) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		a.logger.Info("request",
			"request_id", requestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"response_bytes", recorder.bytes,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func (a *API) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (a *API) liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		a.logger.Warn("readiness check failed", "request_id", requestIDFromContext(r.Context()), "error", err)
		writeProblem(w, r, http.StatusServiceUnavailable, "not-ready", "Service not ready", "ClickHouse is unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func requestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}
