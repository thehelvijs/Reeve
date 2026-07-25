package main

import (
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

type groupView struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

func (a *app) handleListGroups(w http.ResponseWriter, _ *http.Request) {
	groups, err := a.db.ListGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list groups")
		return
	}
	out := make([]groupView, 0, len(groups))
	for _, g := range groups {
		members, err := a.db.ListGroupMembers(g.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not list members")
			return
		}
		if members == nil {
			members = []string{}
		}
		out = append(out, groupView{ID: g.ID, Name: g.Name, Members: members})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "group name is required")
		return
	}
	g, err := a.db.CreateGroup(in.Name)
	if err != nil {
		writeError(w, http.StatusConflict, "name_taken", "a group with that name already exists")
		return
	}
	writeJSON(w, http.StatusCreated, groupView{ID: g.ID, Name: g.Name, Members: []string{}})
}

func (a *app) handleRenameGroup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "group name is required")
		return
	}
	if err := a.db.RenameGroup(id, in.Name); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "group not found")
			return
		}
		writeError(w, http.StatusConflict, "name_taken", "a group with that name already exists")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	if err := a.db.DeleteGroup(r.PathValue("id")); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "group not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not delete group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleAddGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")
	userID := r.PathValue("userId")
	if _, err := a.db.GetUserByID(userID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err := a.db.AddGroupMember(groupID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not add member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	if err := a.db.RemoveGroupMember(r.PathValue("id"), r.PathValue("userId")); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not remove member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
