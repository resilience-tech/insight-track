package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) GetUser(ctx context.Context, userID string) (User, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (User, error) {
		return scanUser(tx.QueryRow(ctx, `
			SELECT id::text, primary_email, primary_email_verified, display_name, avatar_url,
			       status, version, created_at, updated_at
			FROM app.user_account WHERE id = app.current_user_id()
		`))
	})
}

func (s *Store) UpdateUser(ctx context.Context, userID string, version int64, patch UserPatch) (User, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (User, error) {
		user, err := scanUser(tx.QueryRow(ctx, `
			UPDATE app.user_account
			SET display_name = CASE WHEN $2 THEN $3 ELSE display_name END,
			    avatar_url = CASE WHEN $4 THEN $5 ELSE avatar_url END
			WHERE id = app.current_user_id() AND version = $1
			RETURNING id::text, primary_email, primary_email_verified, display_name, avatar_url,
			          status, version, created_at, updated_at
		`, version, patch.DisplayName != nil, patch.DisplayName, patch.AvatarURL.Set, patch.AvatarURL.Value))
		if err == pgx.ErrNoRows {
			return User{}, ErrPrecondition
		}
		return user, err
	})
}

func scanUser(row pgx.Row) (User, error) {
	var value User
	err := row.Scan(&value.ID, &value.PrimaryEmail, &value.PrimaryEmailVerified, &value.DisplayName,
		&value.AvatarURL, &value.Status, &value.Version, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func (s *Store) ListProjects(ctx context.Context, userID, ownership, cursor string, limit int) (Page[Project], error) {
	var after TimeIDCursor
	var afterTime *time.Time
	var afterID *string
	if cursor != "" {
		if err := DecodeCursor(cursor, &after); err != nil || after.Time.IsZero() || after.ID == "" {
			return Page[Project]{}, ErrValidation
		}
		afterTime, afterID = &after.Time, &after.ID
	}
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Page[Project], error) {
		rows, err := tx.Query(ctx, `
			SELECT id::text, owner_user_id::text, name, description,
			       CASE WHEN owner_user_id = app.current_user_id() THEN 'owner' ELSE 'shared' END,
			       version, created_at, updated_at
			FROM app.project
			WHERE ($1 = 'all'
			       OR ($1 = 'owned' AND owner_user_id = app.current_user_id())
			       OR ($1 = 'shared' AND owner_user_id <> app.current_user_id()))
			  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, ($3::text)::uuid))
			ORDER BY created_at DESC, id DESC
			LIMIT $4
		`, ownership, afterTime, afterID, limit+1)
		if err != nil {
			return Page[Project]{}, err
		}
		defer rows.Close()
		items := make([]Project, 0, limit+1)
		for rows.Next() {
			item, err := scanProject(rows)
			if err != nil {
				return Page[Project]{}, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return Page[Project]{}, err
		}
		page := Page[Project]{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			items = items[:limit]
			page.Items = items
			next, err := EncodeCursor(TimeIDCursor{Time: last.CreatedAt, ID: last.ID})
			if err != nil {
				return Page[Project]{}, err
			}
			page.NextCursor = &next
		}
		return page, nil
	})
}

func (s *Store) CreateProject(ctx context.Context, userID, name string, description *string, requestID string) (Project, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Project, error) {
		project, err := scanProject(tx.QueryRow(ctx, `
			INSERT INTO app.project(owner_user_id, name, description)
			VALUES (app.current_user_id(), $1, $2)
			RETURNING id::text, owner_user_id::text, name, description, 'owner', version, created_at, updated_at
		`, name, description))
		if err != nil {
			return Project{}, err
		}
		if err := audit(ctx, tx, project.ID, "project.created", "project", project.ID, requestID, nil); err != nil {
			return Project{}, err
		}
		return project, nil
	})
}

func (s *Store) GetProject(ctx context.Context, userID, projectID string) (Project, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Project, error) {
		return scanProject(tx.QueryRow(ctx, `
			SELECT id::text, owner_user_id::text, name, description,
			       CASE WHEN owner_user_id = app.current_user_id() THEN 'owner' ELSE 'shared' END,
			       version, created_at, updated_at
			FROM app.project WHERE id = ($1::text)::uuid
		`, projectID))
	})
}

func (s *Store) UpdateProject(ctx context.Context, userID, projectID string, version int64, patch ProjectPatch, requestID string) (Project, error) {
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Project, error) {
		project, err := scanProject(tx.QueryRow(ctx, `
			UPDATE app.project
			SET name = CASE WHEN $3 THEN $4 ELSE name END,
			    description = CASE WHEN $5 THEN $6 ELSE description END
			WHERE id = ($1::text)::uuid AND version = $2
			RETURNING id::text, owner_user_id::text, name, description, 'owner', version, created_at, updated_at
		`, projectID, version, patch.Name != nil, patch.Name, patch.Description.Set, patch.Description.Value))
		if err == pgx.ErrNoRows {
			if stateErr := projectMutationState(ctx, tx, userID, projectID, version); stateErr != nil {
				return Project{}, stateErr
			}
		}
		if err != nil {
			return Project{}, err
		}
		if err := audit(ctx, tx, project.ID, "project.updated", "project", project.ID, requestID, nil); err != nil {
			return Project{}, err
		}
		return project, nil
	})
}

func (s *Store) DeleteProject(ctx context.Context, userID, projectID string, version int64) error {
	_, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (struct{}, error) {
		command, err := tx.Exec(ctx, `DELETE FROM app.project WHERE id = ($1::text)::uuid AND version = $2`, projectID, version)
		if err != nil {
			return struct{}{}, err
		}
		if command.RowsAffected() == 0 {
			return struct{}{}, projectMutationState(ctx, tx, userID, projectID, version)
		}
		return struct{}{}, nil
	})
	return err
}

func scanProject(row pgx.Row) (Project, error) {
	var value Project
	err := row.Scan(&value.ID, &value.OwnerUserID, &value.Name, &value.Description, &value.AccessType,
		&value.Version, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func projectMutationState(ctx context.Context, tx pgx.Tx, userID, projectID string, expectedVersion int64) error {
	var ownerID string
	var actualVersion int64
	if err := tx.QueryRow(ctx, `SELECT owner_user_id::text, version FROM app.project WHERE id = ($1::text)::uuid`, projectID).Scan(&ownerID, &actualVersion); err != nil {
		return err
	}
	if ownerID != userID {
		return ErrForbidden
	}
	if actualVersion != expectedVersion {
		return ErrPrecondition
	}
	return ErrConflict
}
