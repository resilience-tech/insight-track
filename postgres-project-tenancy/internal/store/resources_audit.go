package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListResources(ctx context.Context, userID, projectID, serviceID, cursor string, limit int) (Page[Resource], error) {
	var after TimeIDCursor
	var afterTime *time.Time
	var afterID *string
	if cursor != "" {
		if err := DecodeCursor(cursor, &after); err != nil || after.Time.IsZero() || after.ID == "" {
			return Page[Resource]{}, ErrValidation
		}
		afterTime, afterID = &after.Time, &after.ID
	}
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Page[Resource], error) {
		if err := ensureService(ctx, tx, projectID, serviceID); err != nil {
			return Page[Resource]{}, err
		}
		rows, err := tx.Query(ctx, `
			SELECT project_id::text, service_id::text, id::text, resource_key, payload, version, created_at, updated_at
			FROM app.service_resource
			WHERE project_id = ($1::text)::uuid AND service_id = ($2::text)::uuid
			  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3, ($4::text)::uuid))
			ORDER BY created_at DESC, id DESC
			LIMIT $5
		`, projectID, serviceID, afterTime, afterID, limit+1)
		if err != nil {
			return Page[Resource]{}, err
		}
		defer rows.Close()
		items := make([]Resource, 0, limit+1)
		for rows.Next() {
			item, err := scanResource(rows)
			if err != nil {
				return Page[Resource]{}, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return Page[Resource]{}, err
		}
		page := Page[Resource]{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			page.Items = items[:limit]
			next, err := EncodeCursor(TimeIDCursor{Time: last.CreatedAt, ID: last.ID})
			if err != nil {
				return Page[Resource]{}, err
			}
			page.NextCursor = &next
		}
		return page, nil
	})
}

func (s *Store) CreateResource(ctx context.Context, userID, projectID, serviceID, key string, payload json.RawMessage, requestID string) (Resource, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Resource, error) {
		if err := ensureService(ctx, tx, projectID, serviceID); err != nil {
			return Resource{}, err
		}
		resource, err := scanResource(tx.QueryRow(ctx, `
			INSERT INTO app.service_resource(project_id, service_id, resource_key, payload)
			VALUES (($1::text)::uuid, ($2::text)::uuid, $3, $4::jsonb)
			RETURNING project_id::text, service_id::text, id::text, resource_key, payload,
			          version, created_at, updated_at
		`, projectID, serviceID, key, payload))
		if err != nil {
			return Resource{}, err
		}
		if err := audit(ctx, tx, projectID, "resource.created", "service_resource", resource.ID, requestID,
			map[string]any{"service_id": serviceID, "resource_key": key}); err != nil {
			return Resource{}, err
		}
		return resource, nil
	})
}

func (s *Store) GetResource(ctx context.Context, userID, projectID, serviceID, resourceID string) (Resource, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Resource, error) {
		return scanResource(tx.QueryRow(ctx, `
			SELECT project_id::text, service_id::text, id::text, resource_key, payload, version, created_at, updated_at
			FROM app.service_resource
			WHERE project_id = ($1::text)::uuid AND service_id = ($2::text)::uuid
			  AND id = ($3::text)::uuid
		`, projectID, serviceID, resourceID))
	})
}

func (s *Store) UpdateResource(ctx context.Context, userID, projectID, serviceID, resourceID string, version int64, patch ResourcePatch, requestID string) (Resource, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Resource, error) {
		resource, err := scanResource(tx.QueryRow(ctx, `
			UPDATE app.service_resource
			SET resource_key = CASE WHEN $5 THEN $6 ELSE resource_key END,
			    payload = CASE WHEN $7 THEN $8::jsonb ELSE payload END
			WHERE project_id = ($1::text)::uuid AND service_id = ($2::text)::uuid
			  AND id = ($3::text)::uuid AND version = $4
			RETURNING project_id::text, service_id::text, id::text, resource_key, payload,
			          version, created_at, updated_at
		`, projectID, serviceID, resourceID, version,
			patch.ResourceKey != nil, patch.ResourceKey,
			patch.Payload != nil, patch.Payload))
		if err == pgx.ErrNoRows {
			return Resource{}, versionedResourceState(ctx, tx, projectID, serviceID, resourceID, version)
		}
		if err != nil {
			return Resource{}, err
		}
		if err := audit(ctx, tx, projectID, "resource.updated", "service_resource", resource.ID, requestID, nil); err != nil {
			return Resource{}, err
		}
		return resource, nil
	})
}

func (s *Store) DeleteResource(ctx context.Context, userID, projectID, serviceID, resourceID string, version int64, requestID string) error {
	_, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (struct{}, error) {
		command, err := tx.Exec(ctx, `
			DELETE FROM app.service_resource
			WHERE project_id = ($1::text)::uuid AND service_id = ($2::text)::uuid
			  AND id = ($3::text)::uuid AND version = $4
		`, projectID, serviceID, resourceID, version)
		if err != nil {
			return struct{}{}, err
		}
		if command.RowsAffected() == 0 {
			return struct{}{}, versionedResourceState(ctx, tx, projectID, serviceID, resourceID, version)
		}
		if err := audit(ctx, tx, projectID, "resource.deleted", "service_resource", resourceID, requestID,
			map[string]any{"service_id": serviceID}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}

func (s *Store) ListAuditEvents(ctx context.Context, userID, projectID, action, cursor string, limit int) (Page[AuditEvent], error) {
	var after AuditCursor
	var afterTime *time.Time
	var afterID *int64
	if cursor != "" {
		if err := DecodeCursor(cursor, &after); err != nil || after.Time.IsZero() || after.ID < 1 {
			return Page[AuditEvent]{}, ErrValidation
		}
		afterTime, afterID = &after.Time, &after.ID
	}
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Page[AuditEvent], error) {
		if err := ensureMember(ctx, tx, projectID); err != nil {
			return Page[AuditEvent]{}, err
		}
		rows, err := tx.Query(ctx, `
			SELECT id, project_id::text, actor_user_id::text, request_id::text, action, resource_type,
			       resource_id::text, details, occurred_at
			FROM app.project_audit_event
			WHERE project_id = ($1::text)::uuid
			  AND ($2 = '' OR action = $2)
			  AND ($3::timestamptz IS NULL OR (occurred_at, id) < ($3, $4::bigint))
			ORDER BY occurred_at DESC, id DESC
			LIMIT $5
		`, projectID, action, afterTime, afterID, limit+1)
		if err != nil {
			return Page[AuditEvent]{}, err
		}
		defer rows.Close()
		items := make([]AuditEvent, 0, limit+1)
		for rows.Next() {
			var item AuditEvent
			if err := rows.Scan(&item.ID, &item.ProjectID, &item.ActorUserID, &item.RequestID,
				&item.Action, &item.ResourceType, &item.ResourceID, &item.Details, &item.OccurredAt); err != nil {
				return Page[AuditEvent]{}, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return Page[AuditEvent]{}, err
		}
		page := Page[AuditEvent]{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			page.Items = items[:limit]
			next, err := EncodeCursor(AuditCursor{Time: last.OccurredAt, ID: last.ID})
			if err != nil {
				return Page[AuditEvent]{}, err
			}
			page.NextCursor = &next
		}
		return page, nil
	})
}

func scanResource(row pgx.Row) (Resource, error) {
	var value Resource
	err := row.Scan(&value.ProjectID, &value.ServiceID, &value.ID, &value.ResourceKey,
		&value.Payload, &value.Version, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func ensureService(ctx context.Context, tx pgx.Tx, projectID, serviceID string) error {
	var found bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM app.project_service
			WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid
		)
	`, projectID, serviceID).Scan(&found); err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func versionedResourceState(ctx context.Context, tx pgx.Tx, projectID, serviceID, resourceID string, expectedVersion int64) error {
	var actualVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT version FROM app.service_resource
		WHERE project_id = ($1::text)::uuid AND service_id = ($2::text)::uuid
		  AND id = ($3::text)::uuid
	`, projectID, serviceID, resourceID).Scan(&actualVersion); err != nil {
		return err
	}
	if actualVersion != expectedVersion {
		return ErrPrecondition
	}
	return ErrConflict
}
