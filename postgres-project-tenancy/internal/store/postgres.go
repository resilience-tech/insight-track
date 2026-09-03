package store

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var roleNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL, databaseRole string, maxConns, minConns int32, healthPeriod time.Duration) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if databaseRole != "" && !roleNamePattern.MatchString(databaseRole) {
		return nil, fmt.Errorf("DATABASE_ROLE is not a valid PostgreSQL role name")
	}
	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.HealthCheckPeriod = healthPeriod
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	if databaseRole != "" {
		poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, "SET ROLE "+pgx.Identifier{databaseRole}.Sanitize())
			return err
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	store := &Store{pool: pool}
	if err := store.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (s *Store) ResolveIdentity(ctx context.Context, identity Identity) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx, `
		SELECT app.resolve_oidc_user($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), NULLIF($6, ''))::text
	`, identity.Issuer, identity.Subject, identity.Email, identity.EmailVerified, identity.DisplayName, identity.AvatarURL).Scan(&userID)
	if err != nil {
		return "", classify(err)
	}
	return userID, nil
}

func withUserTx[T any](ctx context.Context, s *Store, userID string, fn func(pgx.Tx) (T, error)) (result T, err error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		rollbackContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackContext)
	}()

	if _, err = tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, userID); err != nil {
		return result, fmt.Errorf("set tenant context: %w", err)
	}
	result, err = fn(tx)
	if err != nil {
		return result, classify(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return result, classify(err)
	}
	return result, nil
}

func audit(ctx context.Context, tx pgx.Tx, projectID, action, resourceType, resourceID, requestID string, details any) error {
	if details == nil {
		details = map[string]any{}
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	_, err = tx.Exec(ctx, `
		SELECT app.write_project_audit_event(
			($1::text)::uuid, $2, $3, (NULLIF($4, '')::text)::uuid,
			(NULLIF($5, '')::text)::uuid, $6::jsonb
		)
	`, projectID, action, resourceType, resourceID, requestID, raw)
	return err
}

func ensureMember(ctx context.Context, tx pgx.Tx, projectID string) error {
	var found bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM app.project WHERE id = ($1::text)::uuid)`, projectID).Scan(&found); err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func ensureOwner(ctx context.Context, tx pgx.Tx, userID, projectID string) error {
	var ownerID string
	if err := tx.QueryRow(ctx, `SELECT owner_user_id::text FROM app.project WHERE id = ($1::text)::uuid`, projectID).Scan(&ownerID); err != nil {
		return err
	}
	if ownerID != userID {
		return ErrForbidden
	}
	return nil
}
