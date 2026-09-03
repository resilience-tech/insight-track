package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/example/project-tenancy/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type createServiceRequest struct {
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	Kind          string          `json:"kind"`
	Description   *string         `json:"description"`
	Configuration json.RawMessage `json:"configuration"`
	State         string          `json:"state"`
}

type updateServiceRequest struct {
	Name          optionalString `json:"name"`
	Slug          optionalString `json:"slug"`
	Kind          optionalString `json:"kind"`
	Description   optionalString `json:"description"`
	Configuration optionalJSON   `json:"configuration"`
	State         optionalString `json:"state"`
}

func (a *API) listServices(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	limit, cursor, err := parsePage(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if kind != "" && !validateText(kind, 1, 100) {
		writeValidation(w, r, "kind must contain at most 100 characters.")
		return
	}
	if state != "" && state != "active" && state != "disabled" {
		writeValidation(w, r, "state must be active or disabled.")
		return
	}
	page, err := a.store.ListServices(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), kind, state, cursor, limit)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) createService(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	var body createServiceRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Kind = strings.TrimSpace(body.Kind)
	if !validateText(body.Name, 1, 200) {
		writeValidation(w, r, "name must contain between 1 and 200 characters.")
		return
	}
	if len(body.Slug) < 1 || len(body.Slug) > 63 || !slugPattern.MatchString(body.Slug) {
		writeValidation(w, r, "slug must be 1-63 lowercase letters, numbers, or single hyphen separators.")
		return
	}
	if !validateText(body.Kind, 1, 100) {
		writeValidation(w, r, "kind must contain between 1 and 100 characters.")
		return
	}
	if !validateNullableText(body.Description, 5000) {
		writeValidation(w, r, "description must contain at most 5000 characters.")
		return
	}
	if len(body.Configuration) == 0 {
		body.Configuration = json.RawMessage(`{}`)
	}
	if !validateJSONObject(body.Configuration) {
		writeValidation(w, r, "configuration must be a JSON object.")
		return
	}
	if body.State == "" {
		body.State = "active"
	}
	if body.State != "active" && body.State != "disabled" {
		writeValidation(w, r, "state must be active or disabled.")
		return
	}
	service, err := a.store.CreateService(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"),
		body.Name, body.Slug, body.Kind, body.Description, body.Configuration, body.State, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, service.Version)
	w.Header().Set("Location", "/v1/projects/"+service.ProjectID+"/services/"+service.ID)
	writeJSON(w, http.StatusCreated, service)
}

func (a *API) getService(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId") {
		return
	}
	service, err := a.store.GetService(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, service.Version)
	writeJSON(w, http.StatusOK, service)
}

func (a *API) updateService(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId") {
		return
	}
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	var body updateServiceRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	if !body.Name.Set && !body.Slug.Set && !body.Kind.Set && !body.Description.Set && !body.Configuration.Set && !body.State.Set {
		writeValidation(w, r, "At least one editable field is required.")
		return
	}
	var name, slug, kind, state *string
	if body.Name.Set {
		if body.Name.Value == nil || !validateText(*body.Name.Value, 1, 200) {
			writeValidation(w, r, "name must contain between 1 and 200 characters.")
			return
		}
		value := strings.TrimSpace(*body.Name.Value)
		name = &value
	}
	if body.Slug.Set {
		if body.Slug.Value == nil || len(*body.Slug.Value) < 1 || len(*body.Slug.Value) > 63 || !slugPattern.MatchString(*body.Slug.Value) {
			writeValidation(w, r, "slug must be 1-63 lowercase letters, numbers, or single hyphen separators.")
			return
		}
		slug = body.Slug.Value
	}
	if body.Kind.Set {
		if body.Kind.Value == nil || !validateText(*body.Kind.Value, 1, 100) {
			writeValidation(w, r, "kind must contain between 1 and 100 characters.")
			return
		}
		value := strings.TrimSpace(*body.Kind.Value)
		kind = &value
	}
	if body.Description.Set && !validateNullableText(body.Description.Value, 5000) {
		writeValidation(w, r, "description must contain at most 5000 characters.")
		return
	}
	if body.Configuration.Set && !validateJSONObject(body.Configuration.Value) {
		writeValidation(w, r, "configuration must be a JSON object.")
		return
	}
	if body.State.Set {
		if body.State.Value == nil || (*body.State.Value != "active" && *body.State.Value != "disabled") {
			writeValidation(w, r, "state must be active or disabled.")
			return
		}
		state = body.State.Value
	}
	var configuration *json.RawMessage
	if body.Configuration.Set {
		configuration = &body.Configuration.Value
	}
	service, err := a.store.UpdateService(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"), version,
		store.ServicePatch{
			Name: name, Slug: slug, Kind: kind,
			Description: store.OptionalString{Set: body.Description.Set, Value: body.Description.Value},
			Configuration: configuration, State: state,
		}, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, service.Version)
	writeJSON(w, http.StatusOK, service)
}

func (a *API) deleteService(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId", "serviceId") {
		return
	}
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	err := a.store.DeleteService(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), r.PathValue("serviceId"), version, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
