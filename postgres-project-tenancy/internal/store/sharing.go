package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListMembers(ctx context.Context, userID, projectID, cursor string, limit int) (Page[Member], error) {
	var after TimeIDCursor
	var afterTime *time.Time
	var afterID *string
	if cursor != "" {
		if err := DecodeCursor(cursor, &after); err != nil || after.Time.IsZero() || after.ID == "" {
			return Page[Member]{}, ErrValidation
		}
		afterTime, afterID = &after.Time, &after.ID
	}
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Page[Member], error) {
		rows, err := tx.Query(ctx, `
			SELECT user_id::text, display_name, avatar_url, is_owner, joined_at
			FROM app.list_project_members(($1::text)::uuid, $2, $3, ($4::text)::uuid)
		`, projectID, limit+1, afterTime, afterID)
		if err != nil {
			return Page[Member]{}, err
		}
		defer rows.Close()
		items := make([]Member, 0, limit+1)
		for rows.Next() {
			var item Member
			if err := rows.Scan(&item.UserID, &item.DisplayName, &item.AvatarURL, &item.IsOwner, &item.JoinedAt); err != nil {
				return Page[Member]{}, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return Page[Member]{}, err
		}
		page := Page[Member]{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			page.Items = items[:limit]
			next, err := EncodeCursor(TimeIDCursor{Time: last.JoinedAt, ID: last.UserID})
			if err != nil {
				return Page[Member]{}, err
			}
			page.NextCursor = &next
		}
		return page, nil
	})
}

func (s *Store) RemoveMember(ctx context.Context, userID, projectID, memberID, requestID string) error {
	_, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (struct{}, error) {
		if err := ensureOwner(ctx, tx, userID, projectID); err != nil {
			return struct{}{}, err
		}
		if strings.EqualFold(memberID, userID) {
			return struct{}{}, ErrConflict
		}
		command, err := tx.Exec(ctx, `
			DELETE FROM app.project_member
			WHERE project_id = ($1::text)::uuid AND user_id = ($2::text)::uuid
		`, projectID, memberID)
		if err != nil {
			return struct{}{}, err
		}
		if command.RowsAffected() > 0 {
			if err := audit(ctx, tx, projectID, "member.removed", "user", memberID, requestID, nil); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, nil
	})
	return err
}

func (s *Store) LeaveProject(ctx context.Context, userID, projectID, requestID string) error {
	_, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (struct{}, error) {
		var ownerID string
		if err := tx.QueryRow(ctx, `SELECT owner_user_id::text FROM app.project WHERE id = ($1::text)::uuid`, projectID).Scan(&ownerID); err != nil {
			return struct{}{}, err
		}
		if ownerID == userID {
			return struct{}{}, ErrConflict
		}
		if err := audit(ctx, tx, projectID, "member.left", "user", userID, requestID, nil); err != nil {
			return struct{}{}, err
		}
		command, err := tx.Exec(ctx, `
			DELETE FROM app.project_member
			WHERE project_id = ($1::text)::uuid AND user_id = app.current_user_id()
		`, projectID)
		if err != nil {
			return struct{}{}, err
		}
		if command.RowsAffected() == 0 {
			return struct{}{}, ErrNotFound
		}
		return struct{}{}, nil
	})
	return err
}

func (s *Store) ListInvitations(ctx context.Context, userID, projectID, status, cursor string, limit int) (Page[Invitation], error) {
	var after TimeIDCursor
	var afterTime *time.Time
	var afterID *string
	if cursor != "" {
		if err := DecodeCursor(cursor, &after); err != nil || after.Time.IsZero() || after.ID == "" {
			return Page[Invitation]{}, ErrValidation
		}
		afterTime, afterID = &after.Time, &after.ID
	}
	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (Page[Invitation], error) {
		if err := ensureOwner(ctx, tx, userID, projectID); err != nil {
			return Page[Invitation]{}, err
		}
		rows, err := tx.Query(ctx, `
			SELECT id::text, project_id::text, email,
			       CASE WHEN status = 'pending' AND expires_at <= statement_timestamp() THEN 'expired' ELSE status END,
			       expires_at, accepted_by_user_id::text, accepted_at, created_at
			FROM app.project_invitation
			WHERE project_id = ($1::text)::uuid
			  AND ($2 = '' OR CASE
			         WHEN status = 'pending' AND expires_at <= statement_timestamp() THEN 'expired'
			         ELSE status
			      END = $2)
			  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3, ($4::text)::uuid))
			ORDER BY created_at DESC, id DESC
			LIMIT $5
		`, projectID, status, afterTime, afterID, limit+1)
		if err != nil {
			return Page[Invitation]{}, err
		}
		defer rows.Close()
		items := make([]Invitation, 0, limit+1)
		for rows.Next() {
			item, err := scanInvitation(rows)
			if err != nil {
				return Page[Invitation]{}, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return Page[Invitation]{}, err
		}
		page := Page[Invitation]{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			page.Items = items[:limit]
			next, err := EncodeCursor(TimeIDCursor{Time: last.CreatedAt, ID: last.ID})
			if err != nil {
				return Page[Invitation]{}, err
			}
			page.NextCursor = &next
		}
		return page, nil
	})
}

func (s *Store) CreateInvitation(ctx context.Context, userID, projectID, email string, expiresAt time.Time, requestID string) (CreatedInvitation, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return CreatedInvitation{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	digest := sha256.Sum256([]byte(token))
	email = strings.ToLower(strings.TrimSpace(email))

	return withUserTx(ctx, s, userID, func(tx pgx.Tx) (CreatedInvitation, error) {
		if err := ensureOwner(ctx, tx, userID, projectID); err != nil {
			return CreatedInvitation{}, err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM app.project_invitation
			WHERE project_id = ($1::text)::uuid AND lower(email) = $2 AND status = 'pending'
			  AND expires_at <= statement_timestamp()
		`, projectID, email); err != nil {
			return CreatedInvitation{}, err
		}
		invitation, err := scanInvitation(tx.QueryRow(ctx, `
			INSERT INTO app.project_invitation(project_id, email, token_hash, invited_by_user_id, expires_at)
			VALUES (($1::text)::uuid, $2, $3, app.current_user_id(), $4)
			RETURNING id::text, project_id::text, email, status, expires_at,
			          accepted_by_user_id::text, accepted_at, created_at
		`, projectID, email, digest[:], expiresAt))
		if err != nil {
			return CreatedInvitation{}, err
		}
		if err := audit(ctx, tx, projectID, "invitation.created", "invitation", invitation.ID, requestID,
			map[string]any{"email": email, "expires_at": expiresAt}); err != nil {
			return CreatedInvitation{}, err
		}
		return CreatedInvitation{Invitation: invitation, Token: token}, nil
	})
}

func (s *Store) DeleteInvitation(ctx context.Context, userID, projectID, invitationID, requestID string) error {
	_, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (struct{}, error) {
		if err := ensureOwner(ctx, tx, userID, projectID); err != nil {
			return struct{}{}, err
		}
		var status string
		err := tx.QueryRow(ctx, `
			SELECT status FROM app.project_invitation
			WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid
		`, projectID, invitationID).Scan(&status)
		if err == pgx.ErrNoRows {
			return struct{}{}, nil
		}
		if err != nil {
			return struct{}{}, err
		}
		if status != "pending" {
			return struct{}{}, ErrConflict
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM app.project_invitation
			WHERE project_id = ($1::text)::uuid AND id = ($2::text)::uuid
		`, projectID, invitationID); err != nil {
			return struct{}{}, err
		}
		if err := audit(ctx, tx, projectID, "invitation.revoked", "invitation", invitationID, requestID, nil); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}

func (s *Store) AcceptInvitation(ctx context.Context, userID, token string) (string, bool, error) {
	type result struct {
		projectID string
		created   bool
	}
	value, err := withUserTx(ctx, s, userID, func(tx pgx.Tx) (result, error) {
		var out result
		err := tx.QueryRow(ctx, `SELECT accepted_project_id::text, membership_created FROM app.accept_project_invitation($1)`, token).
			Scan(&out.projectID, &out.created)
		return out, err
	})
	return value.projectID, value.created, err
}

func scanInvitation(row pgx.Row) (Invitation, error) {
	var value Invitation
	err := row.Scan(&value.ID, &value.ProjectID, &value.Email, &value.Status, &value.ExpiresAt,
		&value.AcceptedByUserID, &value.AcceptedAt, &value.CreatedAt)
	return value, err
}
