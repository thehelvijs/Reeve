package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type collectionView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	CreatorID   string `json:"creator_id"`
	IconURL     string `json:"icon_url"`
	ToolCount   int    `json:"tool_count"`
	CanEdit     bool   `json:"can_edit"`
	CreatedAt   string `json:"created_at"`
}

type collectionDetailView struct {
	collectionView
	ToolIDs []string `json:"tool_ids"`
}

type collectionInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

func (a *app) collectionToView(c store.Collection, canEdit bool) collectionView {
	n, _ := a.db.CountCollectionTools(c.ID)
	return collectionView{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Visibility:  c.Visibility,
		CreatorID:   c.CreatorID,
		IconURL:     assetURL("collections", c.ID, "icon", c.IconPath),
		ToolCount:   n,
		CanEdit:     canEdit,
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
	}
}

func (a *app) handleListPublicCollections(w http.ResponseWriter, _ *http.Request) {
	cols, err := a.db.ListPublicCollections()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list collections")
		return
	}
	out := make([]collectionView, 0, len(cols))
	for _, c := range cols {
		out = append(out, a.collectionToView(c, false))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleListCollections(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	cols, err := a.db.ListCollectionsVisibleTo(p.UserID, p.IsAdmin())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list collections")
		return
	}
	out := make([]collectionView, 0, len(cols))
	for _, c := range cols {
		canEdit, _ := a.db.CanEditCollection(p.UserID, p.IsAdmin(), c.ID)
		out = append(out, a.collectionToView(c, canEdit))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	in, ok := decodeCollectionInput(w, r)
	if !ok {
		return
	}
	c, err := a.db.CreateCollection(store.Collection{
		Name: in.Name, Description: in.Description, Visibility: in.Visibility, CreatorID: p.UserID,
	})
	if err != nil {
		writeError(w, http.StatusConflict, "name_taken", "a collection with that name already exists")
		return
	}
	writeJSON(w, http.StatusCreated, a.collectionToView(c, true))
}

func (a *app) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadVisibleCollection(w, r, p)
	if !ok {
		return
	}
	ids, err := a.db.ListCollectionToolIDs(c.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list collection tools")
		return
	}
	canEdit, _ := a.db.CanEditCollection(p.UserID, p.IsAdmin(), c.ID)
	writeJSON(w, http.StatusOK, collectionDetailView{
		collectionView: a.collectionToView(c, canEdit),
		ToolIDs:        ids,
	})
}

func (a *app) handleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadEditableCollection(w, r, p)
	if !ok {
		return
	}
	in, ok := decodeCollectionInput(w, r)
	if !ok {
		return
	}
	c.Name, c.Description = in.Name, in.Description
	// An omitted visibility keeps the stored value rather than blanking it.
	if in.Visibility != "" {
		c.Visibility = in.Visibility
	}
	if err := a.db.UpdateCollection(c); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "collection not found")
			return
		}
		writeError(w, http.StatusConflict, "name_taken", "a collection with that name already exists")
		return
	}
	writeJSON(w, http.StatusOK, a.collectionToView(c, true))
}

func (a *app) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadVisibleCollection(w, r, p)
	if !ok {
		return
	}
	if !p.IsAdmin() && c.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "only the creator or an admin can delete this collection")
		return
	}
	if err := a.db.DeleteCollection(c.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not delete collection")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleAddCollectionTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadEditableCollection(w, r, p)
	if !ok {
		return
	}
	toolID := r.PathValue("toolId")
	if _, err := a.db.GetTool(toolID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return
	}
	if err := a.db.AddCollectionTool(c.ID, toolID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not add tool")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRemoveCollectionTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadEditableCollection(w, r, p)
	if !ok {
		return
	}
	if err := a.db.RemoveCollectionTool(c.ID, r.PathValue("toolId")); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not remove tool")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleListCollectionEditors(w http.ResponseWriter, r *http.Request) {
	a.writeCollectionGrants(w, r, a.db.ListCollectionEditors)
}

func (a *app) handleAddCollectionEditor(w http.ResponseWriter, r *http.Request) {
	a.changeCollectionGrant(w, r, a.db.AddCollectionEditor)
}

func (a *app) handleRemoveCollectionEditor(w http.ResponseWriter, r *http.Request) {
	a.changeCollectionGrant(w, r, a.db.RemoveCollectionEditor)
}

func (a *app) handleListCollectionVisibility(w http.ResponseWriter, r *http.Request) {
	a.writeCollectionGrants(w, r, a.db.ListCollectionVisibility)
}

func (a *app) handleAddCollectionVisibility(w http.ResponseWriter, r *http.Request) {
	a.changeCollectionGrant(w, r, a.db.AddCollectionVisibility)
}

func (a *app) handleRemoveCollectionVisibility(w http.ResponseWriter, r *http.Request) {
	a.changeCollectionGrant(w, r, a.db.RemoveCollectionVisibility)
}

func (a *app) writeCollectionGrants(w http.ResponseWriter, r *http.Request, list func(string) ([]store.VisibilityGrant, error)) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadEditableCollection(w, r, p)
	if !ok {
		return
	}
	grants, err := list(c.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list grants")
		return
	}
	out := make([]visibilityGrantView, 0, len(grants))
	for _, g := range grants {
		out = append(out, visibilityGrantView{PrincipalType: g.PrincipalType, PrincipalID: g.PrincipalID})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) changeCollectionGrant(w http.ResponseWriter, r *http.Request, change func(string, string, string) error) {
	p, _ := rbac.FromContext(r.Context())
	c, ok := a.loadEditableCollection(w, r, p)
	if !ok {
		return
	}
	ptype := r.PathValue("ptype")
	if !validPrincipalType(w, ptype) {
		return
	}
	if err := change(c.ID, ptype, r.PathValue("pid")); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not change grant")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeCollectionInput(w http.ResponseWriter, r *http.Request) (collectionInput, bool) {
	var in collectionInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return collectionInput{}, false
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "collection name is required")
		return collectionInput{}, false
	}
	if !validVisibility(in.Visibility) {
		writeError(w, http.StatusBadRequest, "invalid_visibility", "visibility must be public or restricted")
		return collectionInput{}, false
	}
	return in, true
}

// loadVisibleCollection fetches the collection and 404s if the principal may
// not see it, so a restricted name never leaks.
func (a *app) loadVisibleCollection(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Collection, bool) {
	id := r.PathValue("id")
	ok, _ := a.db.CanSeeCollection(p.UserID, p.IsAdmin(), id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "collection not found")
		return store.Collection{}, false
	}
	c, err := a.db.GetCollection(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "collection not found")
		return store.Collection{}, false
	}
	return c, true
}

// loadEditableCollection additionally requires admin, creator or editor rights.
func (a *app) loadEditableCollection(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Collection, bool) {
	c, ok := a.loadVisibleCollection(w, r, p)
	if !ok {
		return store.Collection{}, false
	}
	can, _ := a.db.CanEditCollection(p.UserID, p.IsAdmin(), c.ID)
	if !can {
		writeError(w, http.StatusForbidden, "forbidden", "only an editor, the creator or an admin can modify this collection")
		return store.Collection{}, false
	}
	return c, true
}
