package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/example/project-tenancy/internal/store"
)

type contextKey struct{}

type Authenticator struct {
	mode     string
	verifier *oidc.IDTokenVerifier
}

func New(ctx context.Context, mode, issuer, audience, jwksURL string) (*Authenticator, error) {
	a := &Authenticator{mode: mode}
	if mode == "dev" {
		return a, nil
	}

	oidcContext := oidc.ClientContext(ctx, &http.Client{Timeout: 5 * time.Second})
	verifierConfig := &oidc.Config{ClientID: audience}
	if jwksURL != "" {
		a.verifier = oidc.NewVerifier(issuer, oidc.NewRemoteKeySet(oidcContext, jwksURL), verifierConfig)
		return a, nil
	}

	provider, err := oidc.NewProvider(oidcContext, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	a.verifier = provider.Verifier(verifierConfig)
	return a, nil
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := a.authenticate(r)
		if err != nil {
			writeUnauthorized(w, r, err.Error())
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, identity)))
	})
}

func IdentityFromContext(ctx context.Context) (store.Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(store.Identity)
	return identity, ok
}

func (a *Authenticator) authenticate(r *http.Request) (store.Identity, error) {
	if a.mode == "dev" {
		return devIdentity(r)
	}

	header := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return store.Identity{}, errors.New("a bearer access token is required")
	}

	token, err := a.verifier.Verify(r.Context(), parts[1])
	if err != nil {
		return store.Identity{}, errors.New("the bearer access token is invalid")
	}
	var claims struct {
		Subject           string `json:"sub"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
		NotBefore         int64  `json:"nbf"`
	}
	if err := token.Claims(&claims); err != nil {
		return store.Identity{}, errors.New("the bearer token claims are invalid")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return store.Identity{}, errors.New("the bearer token has no subject")
	}
	if claims.NotBefore > 0 && time.Now().Add(30*time.Second).Unix() < claims.NotBefore {
		return store.Identity{}, errors.New("the bearer token is not active yet")
	}
	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = strings.TrimSpace(claims.PreferredUsername)
	}
	return store.Identity{
		Issuer:        token.Issuer,
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		DisplayName:   name,
		AvatarURL:     safeHTTPURL(claims.Picture),
	}, nil
}

func safeHTTPURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if parsed.Host == "" || (scheme != "https" && scheme != "http") {
		return ""
	}
	return parsed.String()
}

func devIdentity(r *http.Request) (store.Identity, error) {
	subject := strings.TrimSpace(r.Header.Get("X-Dev-Subject"))
	if subject == "" {
		return store.Identity{}, errors.New("X-Dev-Subject is required in dev auth mode")
	}
	verified := strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Dev-Email-Verified")), "true")
	return store.Identity{
		Issuer:        "urn:project-tenancy:dev",
		Subject:       subject,
		Email:         strings.TrimSpace(r.Header.Get("X-Dev-Email")),
		EmailVerified: verified,
		DisplayName:   strings.TrimSpace(r.Header.Get("X-Dev-Name")),
		AvatarURL:     safeHTTPURL(r.Header.Get("X-Dev-Avatar")),
	}, nil
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="project-tenancy"`)
	w.WriteHeader(http.StatusUnauthorized)
	body := map[string]any{
		"type":   "https://project-tenancy.example/problems/unauthorized",
		"title":  "Authentication required",
		"status": http.StatusUnauthorized,
		"detail": detail,
	}
	if requestID := strings.TrimSpace(r.Header.Get("X-Request-ID")); requestID != "" {
		body["request_id"] = requestID
	}
	_ = json.NewEncoder(w).Encode(body)
}
