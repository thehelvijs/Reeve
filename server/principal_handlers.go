package main

import (
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
)

type principalsView struct {
	Users  []principalUser  `json:"users"`
	Groups []principalGroup `json:"groups"`
}

type principalUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type principalGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// handleListPrincipals gives any signed-in user the names needed to fill a
// visibility or editor picker. Only an admin reads the addresses: a picker needs
// to tell two people apart, which a label does, and a full staff address list is
// worth more to a phisher than it is to the picker.
func (a *app) handleListPrincipals(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	users, err := a.db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list principals")
		return
	}
	groups, err := a.db.ListGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list principals")
		return
	}
	out := principalsView{Users: []principalUser{}, Groups: []principalGroup{}}
	for _, u := range users {
		view := principalUser{ID: u.ID, DisplayName: u.DisplayName, Email: u.Email}
		if view.DisplayName == "" {
			view.DisplayName = localPart(u.Email)
		}
		if !p.IsAdmin() && u.ID != p.UserID {
			view.Email = ""
		}
		out.Users = append(out.Users, view)
	}
	for _, g := range groups {
		out.Groups = append(out.Groups, principalGroup{ID: g.ID, Name: g.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// localPart labels an account that never set a display name, without handing out
// the address itself.
func localPart(email string) string {
	name, _, _ := strings.Cut(email, "@")
	return name
}
