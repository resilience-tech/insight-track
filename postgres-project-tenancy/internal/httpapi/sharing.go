package httpapi

import (
	"net/http"
	"strings"
	"time"
)

func (a *API) listMembers(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	limit, cursor, err := parsePage(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	page, err := a.store.ListMembers(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), cursor, limit)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) removeMember(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "userId") {
		return
	}
	err := a.store.RemoveMember(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("userId"), requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) leaveProject(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	err := a.store.LeaveProject(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listInvitations(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	limit, cursor, err := parsePage(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "pending" && status != "accepted" && status != "expired" {
		writeValidation(w, r, "status must be pending, accepted, or expired.")
		return
	}
	page, err := a.store.ListInvitations(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), status, cursor, limit)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type createInvitationRequest struct {
	Email          string `json:"email"`
	ExpiresInHours *int   `json:"expires_in_hours"`
}

func (a *API) createInvitation(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	var body createInvitationRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	if !validateEmail(body.Email) {
		writeValidation(w, r, "email must be a valid address of at most 320 characters.")
		return
	}
	hours := 168
	if body.ExpiresInHours != nil {
		hours = *body.ExpiresInHours
	}
	if hours < 1 || hours > 720 {
		writeValidation(w, r, "expires_in_hours must be between 1 and 720.")
		return
	}
	created, err := a.store.CreateInvitation(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), body.Email,
		time.Now().UTC().Add(time.Duration(hours)*time.Hour), requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	if a.invitations != nil {
		if err := a.invitations.Deliver(r.Context(), body.Email, created.Token); err != nil {
			a.logger.Error("invitation delivery failed", "request_id", requestIDFromContext(r.Context()), "invitation_id", created.Invitation.ID, "error", err)
		}
	}
	w.Header().Set("Location", "/v1/projects/"+created.Invitation.ProjectID+"/invitations/"+created.Invitation.ID)
	writeJSON(w, http.StatusCreated, created.Invitation)
}

func (a *API) deleteInvitation(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "invitationId") {
		return
	}
	err := a.store.DeleteInvitation(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("invitationId"), requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type acceptInvitationRequest struct {
	Token string `json:"token"`
}

func (a *API) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	var body acceptInvitationRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	if len(body.Token) < 32 || len(body.Token) > 2048 || strings.TrimSpace(body.Token) != body.Token {
		writeValidation(w, r, "token must contain between 32 and 2048 characters.")
		return
	}
	projectID, created, err := a.store.AcceptInvitation(r.Context(), userIDFromContext(r.Context()), body.Token)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": projectID, "membership_created": created})
}
