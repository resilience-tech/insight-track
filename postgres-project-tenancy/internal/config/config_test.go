package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AUTH_MODE", "dev")
	t.Setenv("ALLOW_DEV_AUTH", "true")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("Load error = %v, want DATABASE_URL error", err)
	}
}

func TestLoadGuardsDevelopmentAuthentication(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("AUTH_MODE", "dev")
	t.Setenv("ALLOW_DEV_AUTH", "false")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ALLOW_DEV_AUTH") {
		t.Fatalf("Load error = %v, want ALLOW_DEV_AUTH error", err)
	}
}
