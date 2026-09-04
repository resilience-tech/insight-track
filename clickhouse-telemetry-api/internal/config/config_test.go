package config

import (
	"strings"
	"testing"
)

func validEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("API_TOKEN", "0123456789abcdef0123456789abcdef")
	t.Setenv("CLICKHOUSE_HTTP_URL", "http://localhost:8123")
	t.Setenv("CLICKHOUSE_DATABASE", "insight_track")
	t.Setenv("CLICKHOUSE_USERNAME", "default")
}

func TestLoadDefaults(t *testing.T) {
	validEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ClickHouseDatabase != "insight_track" {
		t.Fatalf("ClickHouseDatabase = %q", cfg.ClickHouseDatabase)
	}
}

func TestLoadRejectsShortToken(t *testing.T) {
	validEnvironment(t)
	t.Setenv("API_TOKEN", "short")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "API_TOKEN") {
		t.Fatalf("Load error = %v, want API_TOKEN error", err)
	}
}

func TestLoadRejectsInvalidClickHouseSettings(t *testing.T) {
	validEnvironment(t)
	t.Setenv("CLICKHOUSE_HTTP_URL", "file:///tmp/clickhouse")
	t.Setenv("CLICKHOUSE_DATABASE", "bad-name")
	_, err := Load()
	if err == nil {
		t.Fatal("Load succeeded with invalid ClickHouse settings")
	}
	for _, expected := range []string{"CLICKHOUSE_HTTP_URL", "CLICKHOUSE_DATABASE"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("Load error = %q, want %s", err, expected)
		}
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	validEnvironment(t)
	t.Setenv("CLICKHOUSE_TIMEOUT", "never")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CLICKHOUSE_TIMEOUT") {
		t.Fatalf("Load error = %v, want CLICKHOUSE_TIMEOUT error", err)
	}
}
