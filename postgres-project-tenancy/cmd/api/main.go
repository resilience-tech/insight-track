package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/project-tenancy/internal/auth"
	"github.com/example/project-tenancy/internal/config"
	"github.com/example/project-tenancy/internal/httpapi"
	"github.com/example/project-tenancy/internal/invite"
	"github.com/example/project-tenancy/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel(cfg.LogLevel)}))

	rootContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	data, err := store.Open(rootContext, cfg.DatabaseURL, cfg.DatabaseRole, cfg.DatabaseMaxConns, cfg.DatabaseMinConns, cfg.DatabaseHealthPeriod)
	if err != nil {
		return err
	}
	defer data.Close()

	authenticator, err := auth.New(rootContext, cfg.AuthMode, cfg.OIDCIssuer, cfg.OIDCAudience, cfg.OIDCJWKSURL)
	if err != nil {
		return err
	}

	var delivery httpapi.InvitationDelivery
	if cfg.InvitationDelivery == "log" {
		delivery = invite.NewLogDelivery(logger, cfg.InvitationBaseURL)
	} else {
		delivery = invite.DisabledDelivery{}
	}

	handler := httpapi.New(data, authenticator, logger, delivery).Handler()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", "address", cfg.HTTPAddr, "auth_mode", cfg.AuthMode)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-rootContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		logger.Info("HTTP server stopping")
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	}
}

func logLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
