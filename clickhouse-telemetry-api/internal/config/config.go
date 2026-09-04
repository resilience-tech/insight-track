package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var databasePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Config struct {
	HTTPAddr           string
	APIToken           string
	ClickHouseURL      string
	ClickHouseDatabase string
	ClickHouseUsername string
	ClickHousePassword string
	ClickHouseTimeout  time.Duration
	ReadHeaderTimeout  time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ShutdownTimeout    time.Duration
	LogLevel           string
}

func Load() (Config, error) {
	var problems []string
	c := Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		APIToken:           strings.TrimSpace(os.Getenv("API_TOKEN")),
		ClickHouseURL:      env("CLICKHOUSE_HTTP_URL", "http://localhost:8123"),
		ClickHouseDatabase: env("CLICKHOUSE_DATABASE", "insight_track"),
		ClickHouseUsername: env("CLICKHOUSE_USERNAME", "default"),
		ClickHousePassword: os.Getenv("CLICKHOUSE_PASSWORD"),
		ClickHouseTimeout:  duration("CLICKHOUSE_TIMEOUT", 10*time.Second, &problems),
		ReadHeaderTimeout:  duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second, &problems),
		ReadTimeout:        duration("HTTP_READ_TIMEOUT", 15*time.Second, &problems),
		WriteTimeout:       duration("HTTP_WRITE_TIMEOUT", 30*time.Second, &problems),
		IdleTimeout:        duration("HTTP_IDLE_TIMEOUT", 60*time.Second, &problems),
		ShutdownTimeout:    duration("SHUTDOWN_TIMEOUT", 10*time.Second, &problems),
		LogLevel:           strings.ToLower(env("LOG_LEVEL", "info")),
	}

	if len(c.APIToken) < 32 {
		problems = append(problems, "API_TOKEN must contain at least 32 characters")
	}
	parsedURL, err := url.Parse(c.ClickHouseURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		problems = append(problems, "CLICKHOUSE_HTTP_URL must be an absolute HTTP(S) URL")
	}
	if !databasePattern.MatchString(c.ClickHouseDatabase) {
		problems = append(problems, "CLICKHOUSE_DATABASE must be a valid ClickHouse identifier")
	}
	if c.ClickHouseUsername == "" {
		problems = append(problems, "CLICKHOUSE_USERNAME must not be empty")
	}
	if c.LogLevel != "debug" && c.LogLevel != "info" && c.LogLevel != "warn" && c.LogLevel != "error" {
		problems = append(problems, "LOG_LEVEL must be debug, info, warn, or error")
	}
	if len(problems) > 0 {
		return Config{}, errors.New(strings.Join(problems, "; "))
	}
	return c, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func duration(name string, fallback time.Duration, problems *[]string) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		*problems = append(*problems, fmt.Sprintf("%s must be a positive duration", name))
		return fallback
	}
	return value
}
