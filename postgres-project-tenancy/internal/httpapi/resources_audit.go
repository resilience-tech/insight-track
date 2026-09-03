package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/example/project-tenancy/internal/store"
)

type createResourceRequest struct {
	ResourceKey string          `json:"resource_key"`
	Payload     json.RawMessage `json:"payload"`
}

type updateResourceRequest struct {
	ResourceKey optionalString `json:"resource_key"`
	Payload     optionalJSON   `json:"payload"`
}

func (a *API) listResources(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId") {
		return
	}
	limit, cursor, err := parsePage(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	page, err := a.store.ListResources(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"), cursor, limit)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) createResource(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId") {
		return
	}
	var body createResourceRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	body.ResourceKey = strings.TrimSpace(body.ResourceKey)
	if !validateText(body.ResourceKey, 1, 200) {
		writeValidation(w, r, "resource_key must contain between 1 and 200 characters.")
		return
	}
	if len(body.Payload) == 0 {
		body.Payload = json.RawMessage(`{}`)
	}
	if !validateJSONObject(body.Payload) {
		writeValidation(w, r, "payload must be a JSON object.")
		return
	}
	resource, err := a.store.CreateResource(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"),
		body.ResourceKey, body.Payload, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, resource.Version)
	w.Header().Set("Location", "/v1/projects/"+resource.ProjectID+"/services/"+resource.ServiceID+"/resources/"+resource.ID)
	writeJSON(w, http.StatusCreated, resource)
}

func (a *API) getResource(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId", "resourceId") {
		return
	}
	resource, err := a.store.GetResource(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"), r.PathValue("resourceId"))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, resource.Version)
	writeJSON(w, http.StatusOK, resource)
}

func (a *API) updateResource(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId", "resourceId") {
		return
	}
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	var body updateResourceRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	if !body.ResourceKey.Set && !body.Payload.Set {
		writeValidation(w, r, "At least one editable field is required.")
		return
	}
	var key *string
	if body.ResourceKey.Set {
		if body.ResourceKey.Value == nil || !validateText(*body.ResourceKey.Value, 1, 200) {
			writeValidation(w, r, "resource_key must contain between 1 and 200 characters.")
			return
		}
		trimmed := strings.TrimSpace(*body.ResourceKey.Value)
		key = &trimmed
	}
	if body.Payload.Set && !validateJSONObject(body.Payload.Value) {
		writeValidation(w, r, "payload must be a JSON object.")
		return
	}
	var payload *json.RawMessage
	if body.Payload.Set {
		payload = &body.Payload.Value
	}
	resource, err := a.store.UpdateResource(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"), r.PathValue("resourceId"), version,
		store.ResourcePatch{ResourceKey: key, Payload: payload}, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, resource.Version)
	writeJSON(w, http.StatusOK, resource)
}

func (a *API) deleteResource(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId", "resourceId") {
		return
	}
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	err := a.store.DeleteResource(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"), r.PathValue("resourceId"), version, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	limit, cursor, err := parsePage(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	if action != "" && !validateText(action, 1, 100) {
		writeValidation(w, r, "action must contain at most 100 characters.")
		return
	}
	page, err := a.store.ListAuditEvents(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), action, cursor, limit)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}
