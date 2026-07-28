package main

import (
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type groupView struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Members []groupMemberView `json:"members"`
}

// groupMemberView carries the member's name with the membership, so a moderator
// can read their own group without also being handed the whole user list.
type groupMemberView struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Role        string `json:"role"`
}

// canManageGroup reports whether the actor may change this group's membership: an
// admin in any group, a moderator in the one they moderate.
func (a *app) canManageGroup(p auth.Principal, groupID string) bool {
	if p.IsAdmin() {
		return true
	}
	mod, err := a.db.IsGroupModerator(groupID, p.UserID)
	return err == nil && mod
}

// requireGroupManager answers 403 and reports false when the actor may not manage
// the group named in the path.
func (a *app) requireGroupManager(w http.ResponseWriter, r *http.Request) (string, bool) {
	p, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	if !a.canManageGroup(p, id) {
		writeError(w, http.StatusForbidden, "forbidden", "only an admin or a moderator of this group can change its members")
		return "", false
	}
	return id, true
}

// handleListGroups returns every group to an admin, and to anyone else the groups
// they moderate. A basic account that moderates nothing gets an empty list rather
// than a 403: the page it feeds is theirs, it is just empty.
func (a *app) handleListGroups(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	var groups []store.Group
	var err error
	if p.IsAdmin() {
		groups, err = a.db.ListGroups()
	} else {
		groups, err = a.db.ListGroupsModeratedBy(p.UserID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list groups")
		return
	}
	users, err := a.db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read users")
		return
	}
	byID := make(map[string]store.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	out := make([]groupView, 0, len(groups))
	for _, g := range groups {
		members, err := a.db.ListGroupMembers(g.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not list members")
			return
		}
		view := groupView{ID: g.ID, Name: g.Name, Members: make([]groupMemberView, 0, len(members))}
		for _, m := range members {
			u := byID[m.UserID]
			view.Members = append(view.Members, groupMemberView{
				UserID:      m.UserID,
				Email:       u.Email,
				DisplayName: u.DisplayName,
				AvatarURL:   a.adminUserView(u).AvatarURL,
				Role:        m.Role,
			})
		}
		out = append(out, view)
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
	writeJSON(w, http.StatusCreated, groupView{ID: g.ID, Name: g.Name, Members: []groupMemberView{}})
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

// handleAddGroupMember adds a member named by id. The admin picker knows ids; the
// moderator flow types an email and goes through handleAddGroupMemberByEmail.
func (a *app) handleAddGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, ok := a.requireGroupManager(w, r)
	if !ok {
		return
	}
	userID := r.PathValue("userId")
	if _, err := a.db.GetUserByID(userID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err := a.db.AddGroupMember(groupID, userID, store.GroupRoleMember); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not add member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAddGroupMemberByEmail adds an existing account by its address. A
// moderator is never handed the user list, so the address is what they have; an
// unknown one is a 404 whether or not that account exists elsewhere.
func (a *app) handleAddGroupMemberByEmail(w http.ResponseWriter, r *http.Request) {
	groupID, ok := a.requireGroupManager(w, r)
	if !ok {
		return
	}
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	role, ok := groupRole(w, in.Role)
	if !ok {
		return
	}
	u, err := a.db.GetUserByEmail(strings.TrimSpace(strings.ToLower(in.Email)))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "no account has that email address")
		return
	}
	if err := a.db.AddGroupMember(groupID, u.ID, role); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not add member")
		return
	}
	// The row may have predated this call with another role, and the caller asked
	// for this one.
	if err := a.db.SetGroupMemberRole(groupID, u.ID, role); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not set the member's role")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSetGroupMemberRole promotes or demotes one member of one group.
func (a *app) handleSetGroupMemberRole(w http.ResponseWriter, r *http.Request) {
	groupID, ok := a.requireGroupManager(w, r)
	if !ok {
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	role, ok := groupRole(w, in.Role)
	if !ok {
		return
	}
	userID := r.PathValue("userId")
	// A moderator demoting themselves would leave a group they can no longer
	// manage, the same trap handleUpdateUser closes for an admin.
	if p, _ := rbac.FromContext(r.Context()); !p.IsAdmin() && userID == p.UserID && role != store.GroupRoleModerator {
		writeError(w, http.StatusBadRequest, "self_lockout", "you cannot give up moderating this group; ask an admin")
		return
	}
	if err := a.db.SetGroupMemberRole(groupID, userID, role); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "that user is not a member of this group")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not set the member's role")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, ok := a.requireGroupManager(w, r)
	if !ok {
		return
	}
	userID := r.PathValue("userId")
	if p, _ := rbac.FromContext(r.Context()); !p.IsAdmin() && userID == p.UserID {
		writeError(w, http.StatusBadRequest, "self_lockout", "you cannot remove yourself from a group you moderate; ask an admin")
		return
	}
	if err := a.db.RemoveGroupMember(groupID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not remove member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// groupRole validates a membership role, defaulting an empty one to member.
func groupRole(w http.ResponseWriter, role string) (string, bool) {
	if role == "" {
		return store.GroupRoleMember, true
	}
	if role != store.GroupRoleModerator && role != store.GroupRoleMember {
		writeError(w, http.StatusBadRequest, "invalid_role", "role must be moderator or member")
		return "", false
	}
	return role, true
}
