package store

import (
	"encoding/json"
	"time"
)

type Identity struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
}

type User struct {
	ID                   string     `json:"id"`
	PrimaryEmail         *string    `json:"primary_email"`
	PrimaryEmailVerified bool       `json:"primary_email_verified"`
	DisplayName          string     `json:"display_name"`
	AvatarURL            *string    `json:"avatar_url"`
	Status               string     `json:"status"`
	Version              int64      `json:"version"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type Project struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	AccessType  string    `json:"access_type"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Member struct {
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	IsOwner     bool      `json:"is_owner"`
	JoinedAt    time.Time `json:"joined_at"`
}

type Invitation struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"project_id"`
	Email            string     `json:"email"`
	Status           string     `json:"status"`
	ExpiresAt        time.Time  `json:"expires_at"`
	AcceptedByUserID *string    `json:"accepted_by_user_id"`
	AcceptedAt       *time.Time `json:"accepted_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type Service struct {
	ProjectID      string          `json:"project_id"`
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	Kind           string          `json:"kind"`
	Description    *string         `json:"description"`
	Configuration  json.RawMessage `json:"configuration"`
	State          string          `json:"state"`
	CreatedByUserID string         `json:"created_by_user_id"`
	Version        int64           `json:"version"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type Resource struct {
	ProjectID   string          `json:"project_id"`
	ServiceID   string          `json:"service_id"`
	ID          string          `json:"id"`
	ResourceKey string          `json:"resource_key"`
	Payload     json.RawMessage `json:"payload"`
	Version     int64           `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type AuditEvent struct {
	ID            int64           `json:"id"`
	ProjectID     string          `json:"project_id"`
	ActorUserID   *string         `json:"actor_user_id"`
	RequestID     *string         `json:"request_id"`
	Action        string          `json:"action"`
	ResourceType  string          `json:"resource_type"`
	ResourceID    *string         `json:"resource_id"`
	Details       json.RawMessage `json:"details"`
	OccurredAt    time.Time       `json:"occurred_at"`
}

type TimeIDCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

type AuditCursor struct {
	Time time.Time `json:"time"`
	ID   int64     `json:"id"`
}

type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

type UserPatch struct {
	DisplayName *string
	AvatarURL   OptionalString
}

type ProjectPatch struct {
	Name        *string
	Description OptionalString
}

type ServicePatch struct {
	Name          *string
	Slug          *string
	Kind          *string
	Description   OptionalString
	Configuration *json.RawMessage
	State         *string
}

type ResourcePatch struct {
	ResourceKey *string
	Payload     *json.RawMessage
}

type OptionalString struct {
	Set   bool
	Value *string
}

type CreatedInvitation struct {
	Invitation Invitation
	Token      string
}
