package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/example/project-tenancy/internal/auth"
	"github.com/example/project-tenancy/internal/store"
)

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	requestIDKey contextKey = "request_id"
)

type InvitationDelivery interface {
	Deliver(ctx context.Context, email, token string) error
}

type API struct {
	store       *store.Store
	auth        *auth.Authenticator
	logger      *slog.Logger
	invitations InvitationDelivery
}

func New(data *store.Store, authenticator *auth.Authenticator, logger *slog.Logger, invitations InvitationDelivery) *API {
	return &API{store: data, auth: authenticator, logger: logger, invitations: invitations}
}

func (a *API) Handler() http.Handler {
	protected := http.NewServeMux()
	protected.HandleFunc("GET /v1/me", a.getMe)
	protected.HandleFunc("PATCH /v1/me", a.updateMe)
	protected.HandleFunc("GET /v1/projects", a.listProjects)
	protected.HandleFunc("POST /v1/projects", a.createProject)
	protected.HandleFunc("GET /v1/projects/{projectId}", a.getProject)
	protected.HandleFunc("PATCH /v1/projects/{projectId}", a.updateProject)
	protected.HandleFunc("DELETE /v1/projects/{projectId}", a.deleteProject)
	protected.HandleFunc("GET /v1/projects/{projectId}/members", a.listMembers)
	protected.HandleFunc("DELETE /v1/projects/{projectId}/members/{userId}", a.removeMember)
	protected.HandleFunc("DELETE /v1/projects/{projectId}/membership", a.leaveProject)
	protected.HandleFunc("GET /v1/projects/{projectId}/invitations", a.listInvitations)
	protected.HandleFunc("POST /v1/projects/{projectId}/invitations", a.createInvitation)
	protected.HandleFunc("DELETE /v1/projects/{projectId}/invitations/{invitationId}", a.deleteInvitation)
	protected.HandleFunc("POST /v1/invitations/accept", a.acceptInvitation)
	protected.HandleFunc("GET /v1/projects/{projectId}/services", a.listServices)
	protected.HandleFunc("POST /v1/projects/{projectId}/services", a.createService)
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}", a.getService)
	protected.HandleFunc("PATCH /v1/projects/{projectId}/services/{serviceId}", a.updateService)
	protected.HandleFunc("DELETE /v1/projects/{projectId}/services/{serviceId}", a.deleteService)
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}/resources", a.listResources)
	protected.HandleFunc("POST /v1/projects/{projectId}/services/{serviceId}/resources", a.createResource)
	protected.HandleFunc("GET /v1/projects/{projectId}/services/{serviceId}/resources/{resourceId}", a.getResource)
	protected.HandleFunc("PATCH /v1/projects/{projectId}/services/{serviceId}/resources/{resourceId}", a.updateResource)
	protected.HandleFunc("DELETE /v1/projects/{projectId}/services/{serviceId}/resources/{resourceId}", a.deleteResource)
	protected.HandleFunc("GET /v1/projects/{projectId}/audit-events", a.listAuditEvents)

	root := http.NewServeMux()
	root.HandleFunc("GET /health/live", a.liveness)
	root.HandleFunc("GET /health/ready", a.readiness)
	root.Handle("/v1/", a.auth.Middleware(a.resolveIdentity(protected)))

	return a.requestID(a.recoverPanic(a.accessLog(a.securityHeaders(root))))
}

func (a *API) resolveIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := auth.IdentityFromContext(r.Context())
		if !ok {
			writeProblem(w, http.StatusUnauthorized, "unauthorized", "Authentication required", "No authenticated identity was found.", requestIDFromContext(r.Context()))
			return
		}
		userID, err := a.store.ResolveIdentity(r.Context(), identity)
		if err != nil {
			a.writeStoreError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	})
}

func (a *API) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if !validateUUID(requestID) {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
	})
}

func (a *API) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", "request_id", requestIDFromContext(r.Context()), "panic", recovered, "stack", string(debug.Stack()))
				writeProblem(w, http.StatusInternalServerError, "internal-error", "Internal server error", "An unexpected error occurred.", requestIDFromContext(r.Context()))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
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
	return r.ResponseWriter.Write(body)
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
		a.logger.Info("request", "request_id", requestIDFromContext(r.Context()), "method", r.Method,
			"path", r.URL.Path, "status", status, "duration_ms", time.Since(started).Milliseconds())
	})
}

func (a *API) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
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
		writeProblem(w, http.StatusServiceUnavailable, "not-ready", "Service not ready", "The database is unavailable.", requestIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func userIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userIDKey).(string)
	return value
}

func requestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func validatePathIDs(w http.ResponseWriter, r *http.Request, names ...string) bool {
	for _, name := range names {
		if !validateUUID(r.PathValue(name)) {
			writeProblem(w, http.StatusUnprocessableEntity, "validation-error", "Request validation failed", fmt.Sprintf("%s must be a UUID.", name), requestIDFromContext(r.Context()))
			return false
		}
	}
	return true
}

func writeValidation(w http.ResponseWriter, r *http.Request, detail string) {
	writeProblem(w, http.StatusUnprocessableEntity, "validation-error", "Request validation failed", detail, requestIDFromContext(r.Context()))
}

func requireIfMatch(w http.ResponseWriter, r *http.Request) (int64, bool) {
	version, err := parseIfMatch(r)
	if err != nil {
		writeProblem(w, http.StatusPreconditionFailed, "precondition-failed", "If-Match required", err.Error(), requestIDFromContext(r.Context()))
		return 0, false
	}
	return version, true
}
