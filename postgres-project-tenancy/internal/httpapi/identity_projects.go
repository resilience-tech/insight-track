package httpapi

import (
	"net/http"
	"strings"

	"github.com/example/project-tenancy/internal/store"
)

type updateUserRequest struct {
	DisplayName optionalString `json:"display_name"`
	AvatarURL   optionalString `json:"avatar_url"`
}

func (a *API) getMe(w http.ResponseWriter, r *http.Request) {
	user, err := a.store.GetUser(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, user.Version)
	writeJSON(w, http.StatusOK, user)
}

func (a *API) updateMe(w http.ResponseWriter, r *http.Request) {
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	var body updateUserRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	if !body.DisplayName.Set && !body.AvatarURL.Set {
		writeValidation(w, r, "At least one editable field is required.")
		return
	}
	if body.DisplayName.Set && (body.DisplayName.Value == nil || !validateText(*body.DisplayName.Value, 1, 200)) {
		writeValidation(w, r, "display_name must contain between 1 and 200 characters.")
		return
	}
	if body.AvatarURL.Set && !validateURL(body.AvatarURL.Value) {
		writeValidation(w, r, "avatar_url must be null or an absolute HTTP(S) URL.")
		return
	}
	var name *string
	if body.DisplayName.Set {
		trimmed := strings.TrimSpace(*body.DisplayName.Value)
		name = &trimmed
	}
	user, err := a.store.UpdateUser(r.Context(), userIDFromContext(r.Context()), version, store.UserPatch{
		DisplayName: name,
		AvatarURL:   store.OptionalString{Set: body.AvatarURL.Set, Value: body.AvatarURL.Value},
	})
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, user.Version)
	writeJSON(w, http.StatusOK, user)
}

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateProjectRequest struct {
	Name        optionalString `json:"name"`
	Description optionalString `json:"description"`
}

func (a *API) listProjects(w http.ResponseWriter, r *http.Request) {
	limit, cursor, err := parsePage(r)
	if err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	ownership := strings.TrimSpace(r.URL.Query().Get("ownership"))
	if ownership == "" {
		ownership = "all"
	}
	if ownership != "all" && ownership != "owned" && ownership != "shared" {
		writeValidation(w, r, "ownership must be all, owned, or shared.")
		return
	}
	page, err := a.store.ListProjects(r.Context(), userIDFromContext(r.Context()), ownership, cursor, limit)
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) createProject(w http.ResponseWriter, r *http.Request) {
	var body createProjectRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if !validateText(body.Name, 1, 200) {
		writeValidation(w, r, "name must contain between 1 and 200 characters.")
		return
	}
	if !validateNullableText(body.Description, 5000) {
		writeValidation(w, r, "description must contain at most 5000 characters.")
		return
	}
	project, err := a.store.CreateProject(r.Context(), userIDFromContext(r.Context()), body.Name, body.Description, requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, project.Version)
	w.Header().Set("Location", "/v1/projects/"+project.ID)
	writeJSON(w, http.StatusCreated, project)
}

func (a *API) getProject(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	project, err := a.store.GetProject(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, project.Version)
	writeJSON(w, http.StatusOK, project)
}

func (a *API) updateProject(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	var body updateProjectRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeValidation(w, r, err.Error())
		return
	}
	if !body.Name.Set && !body.Description.Set {
		writeValidation(w, r, "At least one editable field is required.")
		return
	}
	var name *string
	if body.Name.Set {
		if body.Name.Value == nil || !validateText(*body.Name.Value, 1, 200) {
			writeValidation(w, r, "name must contain between 1 and 200 characters.")
			return
		}
		trimmed := strings.TrimSpace(*body.Name.Value)
		name = &trimmed
	}
	if body.Description.Set && !validateNullableText(body.Description.Value, 5000) {
		writeValidation(w, r, "description must contain at most 5000 characters.")
		return
	}
	project, err := a.store.UpdateProject(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), version,
		store.ProjectPatch{Name: name, Description: store.OptionalString{Set: body.Description.Set, Value: body.Description.Value}},
		requestIDFromContext(r.Context()))
	if err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	setVersionHeaders(w, project.Version)
	writeJSON(w, http.StatusOK, project)
}

func (a *API) deleteProject(w http.ResponseWriter, r *http.Request) {
	if !validatePathIDs(w, r, "projectId") {
		return
	}
	version, ok := requireIfMatch(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteProject(r.Context(), userIDFromContext(r.Context()), r.PathValue("projectId"), version); err != nil {
		a.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
