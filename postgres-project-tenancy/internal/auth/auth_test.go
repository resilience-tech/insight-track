package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevAuthentication(t *testing.T) {
	t.Parallel()
	authenticator, err := New(context.Background(), "dev", "", "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var subject string
	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		if !ok {
			t.Fatal("identity missing from context")
		}
		subject = identity.Subject
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("X-Dev-Subject", "alice")
	request.Header.Set("X-Dev-Email", "alice@example.test")
	request.Header.Set("X-Dev-Email-Verified", "true")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if subject != "alice" {
		t.Fatalf("subject = %q, want alice", subject)
	}
}

func TestDevAuthenticationRequiresSubject(t *testing.T) {
	t.Parallel()
	authenticator, err := New(context.Background(), "dev", "", "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	handler := authenticator.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSafeHTTPURL(t *testing.T) {
	t.Parallel()
	if got := safeHTTPURL("https://example.test/avatar.png"); got != "https://example.test/avatar.png" {
		t.Fatalf("safeHTTPURL = %q", got)
	}
	for _, value := range []string{"javascript:alert(1)", "data:text/plain,test", "/relative"} {
		if got := safeHTTPURL(value); got != "" {
			t.Fatalf("safeHTTPURL(%q) = %q, want empty", value, got)
		}
	}
}
