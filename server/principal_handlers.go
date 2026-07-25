package main

import "net/http"

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
// visibility or editor picker; it exposes nothing beyond name and email.
func (a *app) handleListPrincipals(w http.ResponseWriter, _ *http.Request) {
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
		out.Users = append(out.Users, principalUser{ID: u.ID, DisplayName: u.DisplayName, Email: u.Email})
	}
	for _, g := range groups {
		out.Groups = append(out.Groups, principalGroup{ID: g.ID, Name: g.Name})
	}
	writeJSON(w, http.StatusOK, out)
}
