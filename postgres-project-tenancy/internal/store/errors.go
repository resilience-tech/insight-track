package store

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrForbidden    = errors.New("operation is not permitted")
	ErrConflict     = errors.New("resource state conflict")
	ErrPrecondition = errors.New("resource version does not match")
	ErrValidation   = errors.New("invalid data")
)

func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "P0002":
		return ErrNotFound
	case "42501":
		switch pgErr.Message {
		case "account_disabled", "authentication_required", "invitation_email_mismatch", "project_access_denied":
			return ErrForbidden
		default:
			return err
		}
	case "23505", "55000":
		return ErrConflict
	case "22023", "23502", "23503", "23514", "22P02":
		return ErrValidation
	default:
		if strings.HasPrefix(pgErr.Code, "23") {
			return ErrConflict
		}
		return err
	}
}
