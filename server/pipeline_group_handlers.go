package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// maxPipelineGroupProjects bounds one group, because every member costs a field
// in the GraphQL call that reads a GitLab group, and a request of its own on
// GitHub.
const maxPipelineGroupProjects = 200

type pipelineGroupRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Provider is the forge the paths belong to: "gitlab" or "github".
	Provider string `json:"provider"`
	// Projects are full paths: "group/subgroup/project" on GitLab,
	// "owner/repo" on GitHub.
	Projects []string `json:"projects"`
}

func pipelineGroupToRecord(g store.PipelineGroup) pipelineGroupRecord {
	return pipelineGroupRecord{ID: g.ID, Name: g.Name, Provider: g.Provider, Projects: g.Projects}
}

// handleListPipelineGroups returns the groups and their members without calling
// GitLab, which is what the manage page edits and the pipelines page falls back
// to while its own request is in flight.
func (a *app) handleListPipelineGroups(w http.ResponseWriter, _ *http.Request) {
	groups, err := a.db.ListPipelineGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list pipeline groups")
		return
	}
	out := make([]pipelineGroupRecord, 0, len(groups))
	for _, g := range groups {
		out = append(out, pipelineGroupToRecord(g))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreatePipelineGroup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "a group needs a name")
		return
	}
	if !validProvider(in.Provider) {
		writeError(w, http.StatusBadRequest, "invalid_provider", "provider must be gitlab or github")
		return
	}
	g, err := a.db.CreatePipelineGroup(name, in.Provider)
	if err != nil {
		writeError(w, http.StatusConflict, "name_taken", "a group with that name already exists")
		return
	}
	writeJSON(w, http.StatusCreated, pipelineGroupToRecord(g))
}

func (a *app) handleRenamePipelineGroup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "a group needs a name")
		return
	}
	if err := a.db.RenamePipelineGroup(r.PathValue("id"), name); err != nil {
		writePipelineGroupError(w, err, "could not rename the group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleDeletePipelineGroup(w http.ResponseWriter, r *http.Request) {
	if err := a.db.DeletePipelineGroup(r.PathValue("id")); err != nil {
		writePipelineGroupError(w, err, "could not delete the group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAddPipelineGroupProject puts one project in a group. The path rides in
// the body rather than the URL because a GitLab full path carries slashes of its
// own.
func (a *app) handleAddPipelineGroupProject(w http.ResponseWriter, r *http.Request) {
	id, path, ok := a.pipelineGroupProjectRequest(w, r)
	if !ok {
		return
	}
	n, err := a.db.CountPipelineGroupProjects(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read the group")
		return
	}
	if n >= maxPipelineGroupProjects {
		writeError(w, http.StatusBadRequest, "group_full",
			"a group holds at most 200 projects; split it into two")
		return
	}
	if err := a.db.AddPipelineGroupProject(id, path); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not add the project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRemovePipelineGroupProject(w http.ResponseWriter, r *http.Request) {
	id, path, ok := a.pipelineGroupProjectRequest(w, r)
	if !ok {
		return
	}
	if err := a.db.RemovePipelineGroupProject(id, path); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not remove the project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pipelineGroupProjectRequest reads and checks the group and project a
// membership write names, answering the client itself on anything wrong.
func (a *app) pipelineGroupProjectRequest(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	var in struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return "", "", false
	}
	path := strings.Trim(strings.TrimSpace(in.Path), "/")
	if path == "" || !strings.Contains(path, "/") {
		writeError(w, http.StatusBadRequest, "invalid_path",
			"a project path looks like group/project")
		return "", "", false
	}
	id := r.PathValue("id")
	exists, err := a.db.PipelineGroupExists(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read the group")
		return "", "", false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "not_found", "no such pipeline group")
		return "", "", false
	}
	return id, path, true
}

func writePipelineGroupError(w http.ResponseWriter, err error, msg string) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "no such pipeline group")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal", msg)
}
