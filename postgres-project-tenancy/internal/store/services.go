package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListServices(ctx context.Context, userID, projectID, kind, state, cursor string, limit int) (Page[Service], error) {
	var after TimeIDCursor
	var afterTime *time.Time
	var afterID *string
	if cursor != "" {
		if err := DecodeCursor(cursor, &after); err != nil || after.Time.IsZero() || after.ID == "" {
			return Page[Service]{}, ErrValidation
		}
		afterTime, afterID = &after.Time, &after.ID
	}
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Page[Service], error) {
		if err := ensureMember(ctx, tx, projectID); err != nil {
			return Page[Service]{}, err
		}
		rows, err := tx.Query(ctx, `
			SELECT project_id::text, id::text, name, slug, kind, description, configuration, state,
			       created_by_user_id::text, version, created_at, updated_at
			FROM app.project_service
			WHERE project_id = ($1::text)::uuid
			  AND ($2 = '' OR kind = $2)
			  AND ($3 = '' OR state = $3)
			  AND ($4::timestamptz IS NULL OR (created_at, id) < ($4, ($5::text)::uuid))
			ORDER BY created_at DESC, id DESC
			LIMIT $6
		`, projectID, kind, state, afterTime, afterID, limit+1)
		if err != nil {
			return Page[Service]{}, err
		}
		defer rows.Close()
		items := make([]Service, 0, limit+1)
		for rows.Next() {
			item, err := scanService(rows)
			if err != nil {
				return Page[Service]{}, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return Page[Service]{}, err
		}
		page := Page[Service]{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			page.Items = items[:limit]
			next, err := EncodeCursor(TimeIDCursor{Time: last.CreatedAt, ID: last.ID})
			if err != nil {
				return Page[Service]{}, err
			}
			page.NextCursor = &next
		}
		return page, nil
	})
}

func (s *Store) CreateService(ctx context.Context, userID, projectID, name, slug, kind string, description *string, configuration json.RawMessage, state, requestID string) (Service, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Service, error) {
		if err := ensureMember(ctx, tx, projectID); err != nil {
			return Service{}, err
		}
		service, err := scanService(tx.QueryRow(ctx, `
			INSERT INTO app.project_service(
				project_id, name, slug, kind, description, configuration, state, created_by_user_id
			)
			VALUES (($1::text)::uuid, $2, $3, $4, $5, $6::jsonb, $7, app.current_user_id())
			RETURNING project_id::text, id::text, name, slug, kind, description, configuration, state,
			          created_by_user_id::text, version, created_at, updated_at
		`, projectID, name, slug, kind, description, configuration, state))
		if err != nil {
			return Service{}, err
		}
		if err := audit(ctx, tx, projectID, "service.created", "service", service.ID, requestID, map[string]any{"slug": slug}); err != nil {
			return Service{}, err
		}
		return service, nil
	})
}

func (s *Store) GetService(ctx context.Context, userID, projectID, serviceID string) (Service, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Service, error) {
		return scanService(tx.QueryRow(ctx, `
			SELECT project_id::text, id::text, name, slug, kind, description, configuration, state,
			       created_by_user_id::text, version, created_at, updated_at
			FROM app.project_service
			WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid
		`, projectID, serviceID))
	})
}

func (s *Store) UpdateService(ctx context.Context, userID, projectID, serviceID string, version int64, patch ServicePatch, requestID string) (Service, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Service, error) {
		service, err := scanService(tx.QueryRow(ctx, `
			UPDATE app.project_service
			SET name = CASE WHEN $4 THEN $5 ELSE name END,
			    slug = CASE WHEN $6 THEN $7 ELSE slug END,
			    kind = CASE WHEN $8 THEN $9 ELSE kind END,
			    description = CASE WHEN $10 THEN $11 ELSE description END,
			    configuration = CASE WHEN $12 THEN $13::jsonb ELSE configuration END,
			    state = CASE WHEN $14 THEN $15 ELSE state END
			WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid AND version = $3
			RETURNING project_id::text, id::text, name, slug, kind, description, configuration, state,
			          created_by_user_id::text, version, created_at, updated_at
		`, projectID, serviceID, version,
			patch.Name != nil, patch.Name,
			patch.Slug != nil, patch.Slug,
			patch.Kind != nil, patch.Kind,
			patch.Description.Set, patch.Description.Value,
			patch.Configuration != nil, patch.Configuration,
			patch.State != nil, patch.State))
		if err == pgx.ErrNoRows {
			return Service{}, versionedServiceState(ctx, tx, projectID, serviceID, version)
		}
		if err != nil {
			return Service{}, err
		}
		if err := audit(ctx, tx, projectID, "service.updated", "service", service.ID, requestID, nil); err != nil {
			return Service{}, err
		}
		return service, nil
	})
}

func (s *Store) DeleteService(ctx context.Context, userID, projectID, serviceID string, version int64, requestID string) error {
	_, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (struct{}, error) {
		command, err := tx.Exec(ctx, `
			DELETE FROM app.project_service
			WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid AND version = $3
		`, projectID, serviceID, version)
		if err != nil {
			return struct{}{}, err
		}
		if command.RowsAffected() == 0 {
			return struct{}{}, versionedServiceState(ctx, tx, projectID, serviceID, version)
		}
		if err := audit(ctx, tx, projectID, "service.deleted", "service", serviceID, requestID, nil); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}

func scanService(row pgx.Row) (Service, error) {
	var value Service
	err := row.Scan(&value.ProjectID, &value.ID, &value.Name, &value.Slug, &value.Kind, &value.Description,
		&value.Configuration, &value.State, &value.CreatedByUserID, &value.Version, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func versionedServiceState(ctx context.Context, tx pgx.Tx, projectID, serviceID string, expectedVersion int64) error {
	var actualVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT version FROM app.project_service
		WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid
	`, projectID, serviceID).Scan(&actualVersion); err != nil {
		return err
	}
	if actualVersion != expectedVersion {
		return ErrPrecondition
	}
	return ErrConflict
}
