package invite

import (
	"context"
	"errors"
	"log/slog"
)

type LogDelivery struct {
	logger  *slog.Logger
	baseURL string
}

func NewLogDelivery(logger *slog.Logger, baseURL string) *LogDelivery {
	return &LogDelivery{logger: logger, baseURL: baseURL}
}

func (d *LogDelivery) Deliver(_ context.Context, email, token string) error {
	if d.logger == nil {
		return errors.New("invitation logger is unavailable")
	}
	d.logger.Warn("development invitation token; do not use log delivery in production",
		"email", email, "accept_page", d.baseURL, "token", token)
	return nil
}

type DisabledDelivery struct{}

func (DisabledDelivery) Deliver(context.Context, string, string) error {
	return errors.New("invitation delivery is disabled")
}
