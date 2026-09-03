package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	DatabaseRole         string
	AuthMode             string
	AllowDevAuth         bool
	OIDCIssuer           string
	OIDCAudience         string
	OIDCJWKSURL          string
	InvitationBaseURL    string
	InvitationDelivery   string
	LogLevel             string
	ReadTimeout          time.Duration
	ReadHeaderTimeout    time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	ShutdownTimeout      time.Duration
	DatabaseMaxConns     int32
	DatabaseMinConns     int32
	DatabaseHealthPeriod time.Duration
}

func Load() (Config, error) {
	var problems []string
	c := Config{
		HTTPAddr:             env("HTTP_ADDR", ":8080"),
		DatabaseURL:          strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DatabaseRole:         env("DATABASE_ROLE", "app_runtime"),
		AuthMode:             strings.ToLower(env("AUTH_MODE", "oidc")),
		AllowDevAuth:         strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_DEV_AUTH")), "true"),
		OIDCIssuer:           strings.TrimSpace(os.Getenv("OIDC_ISSUER")),
		OIDCAudience:         strings.TrimSpace(os.Getenv("OIDC_AUDIENCE")),
		OIDCJWKSURL:          strings.TrimSpace(os.Getenv("OIDC_JWKS_URL")),
		InvitationBaseURL:    strings.TrimRight(env("INVITATION_BASE_URL", "http://localhost:3000/invitations/accept"), "/"),
		InvitationDelivery:   strings.ToLower(env("INVITATION_DELIVERY", "log")),
		LogLevel:             strings.ToLower(env("LOG_LEVEL", "info")),
		ReadTimeout:          duration("HTTP_READ_TIMEOUT", 10*time.Second, &problems),
		ReadHeaderTimeout:    duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second, &problems),
		WriteTimeout:         duration("HTTP_WRITE_TIMEOUT", 30*time.Second, &problems),
		IdleTimeout:          duration("HTTP_IDLE_TIMEOUT", 60*time.Second, &problems),
		ShutdownTimeout:      duration("SHUTDOWN_TIMEOUT", 10*time.Second, &problems),
		DatabaseMaxConns:     int32Value("DATABASE_MAX_CONNS", 20, &problems),
		DatabaseMinConns:     int32Value("DATABASE_MIN_CONNS", 2, &problems),
		DatabaseHealthPeriod: duration("DATABASE_HEALTH_PERIOD", 30*time.Second, &problems),
	}

	if c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if c.AuthMode != "oidc" && c.AuthMode != "dev" {
		problems = append(problems, "AUTH_MODE must be oidc or dev")
	}
	if c.AuthMode == "dev" && !c.AllowDevAuth {
		problems = append(problems, "ALLOW_DEV_AUTH=true is required when AUTH_MODE=dev")
	}
	if c.AuthMode == "oidc" {
		if c.OIDCIssuer == "" {
			problems = append(problems, "OIDC_ISSUER is required in oidc mode")
		}
		if c.OIDCAudience == "" {
			problems = append(problems, "OIDC_AUDIENCE is required in oidc mode")
		}
	}
	if c.InvitationDelivery != "log" && c.InvitationDelivery != "disabled" {
		problems = append(problems, "INVITATION_DELIVERY must be log or disabled")
	}
	if c.LogLevel != "debug" && c.LogLevel != "info" && c.LogLevel != "warn" && c.LogLevel != "error" {
		problems = append(problems, "LOG_LEVEL must be debug, info, warn, or error")
	}
	if c.DatabaseMaxConns < 1 || c.DatabaseMinConns > c.DatabaseMaxConns {
		problems = append(problems, "DATABASE_MAX_CONNS must be positive and at least DATABASE_MIN_CONNS")
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

func int32Value(name string, fallback int32, problems *[]string) int32 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		*problems = append(*problems, fmt.Sprintf("%s must be a non-negative integer", name))
		return fallback
	}
	return int32(value)
}
