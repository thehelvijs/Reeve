package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// iconDir returns the directory that holds tool/host icon files, defaulting to
// an "icons" folder beside the database file when unconfigured.
func (a *app) iconDir() string {
	if a.cfg.IconDir != "" {
		return a.cfg.IconDir
	}
	return filepath.Join(filepath.Dir(a.cfg.DBPath), "icons")
}

// --- tool icons ---------------------------------------------------------

func (a *app) handleUploadToolIcon(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	t, err := a.db.GetTool(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "not allowed to edit this tool")
		return
	}
	data, ext, ok := readImageUpload(w, r, "icon")
	if !ok {
		return
	}
	dest, err := writeImageFile(a.iconDir(), "tool-"+id, ext, data, t.IconPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store icon")
		return
	}
	if err := a.db.SetToolIconPath(id, dest); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store icon")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"icon_url": iconURL("tools", id, dest)})
}

func (a *app) handleDeleteToolIcon(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	t, err := a.db.GetTool(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "not allowed to edit this tool")
		return
	}
	if t.IconPath != "" {
		os.Remove(t.IconPath)
		if err := a.db.SetToolIconPath(id, ""); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not update icon")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"icon_url": ""})
}

// handleServeToolIcon serves a tool's icon, respecting visibility: public tools
// to anyone, restricted tools only to principals who may see them.
func (a *app) handleServeToolIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.db.GetTool(id)
	if err != nil || t.IconPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "no icon")
		return
	}
	visible := t.Visibility == store.VisibilityPublic
	if p, ok := rbac.FromContext(r.Context()); ok {
		if seen, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), id); seen {
			visible = true
		}
	}
	if !visible {
		writeError(w, http.StatusNotFound, "not_found", "no icon")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, t.IconPath)
}

// --- host icons ---------------------------------------------------------

func (a *app) handleUploadHostIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	data, ext, ok := readImageUpload(w, r, "icon")
	if !ok {
		return
	}
	dest, err := writeImageFile(a.iconDir(), "host-"+id, ext, data, h.IconPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store icon")
		return
	}
	if err := a.db.SetHostIconPath(id, dest); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store icon")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"icon_url": iconURL("hosts", id, dest)})
}

func (a *app) handleDeleteHostIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if h.IconPath != "" {
		os.Remove(h.IconPath)
		if err := a.db.SetHostIconPath(id, ""); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not update icon")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"icon_url": ""})
}

// handleServeHostIcon serves a host's icon to any authenticated caller, and to
// anonymous callers only when the host backs a public tool.
func (a *app) handleServeHostIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil || h.IconPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "no icon")
		return
	}
	if _, ok := rbac.FromContext(r.Context()); !ok && !a.db.HostHasPublicTool(id) {
		writeError(w, http.StatusNotFound, "not_found", "no icon")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, h.IconPath)
}

// --- tool thumbnails ----------------------------------------------------

func (a *app) handleUploadToolThumbnail(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	t, err := a.db.GetTool(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "not allowed to edit this tool")
		return
	}
	data, ext, ok := readImageUpload(w, r, "icon")
	if !ok {
		return
	}
	dest, err := writeImageFile(a.iconDir(), "tool-thumb-"+id, ext, data, t.ThumbnailPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store thumbnail")
		return
	}
	if err := a.db.SetToolThumbnailPath(id, dest); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store thumbnail")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"thumbnail_url": thumbnailURL("tools", id, dest)})
}

func (a *app) handleDeleteToolThumbnail(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	id := r.PathValue("id")
	t, err := a.db.GetTool(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "not allowed to edit this tool")
		return
	}
	if t.ThumbnailPath != "" {
		os.Remove(t.ThumbnailPath)
		if err := a.db.SetToolThumbnailPath(id, ""); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not update thumbnail")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"thumbnail_url": ""})
}

func (a *app) handleServeToolThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.db.GetTool(id)
	if err != nil || t.ThumbnailPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "no thumbnail")
		return
	}
	visible := t.Visibility == store.VisibilityPublic
	if p, ok := rbac.FromContext(r.Context()); ok {
		if seen, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), id); seen {
			visible = true
		}
	}
	if !visible {
		writeError(w, http.StatusNotFound, "not_found", "no thumbnail")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, t.ThumbnailPath)
}

// --- host thumbnails ----------------------------------------------------

func (a *app) handleUploadHostThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	data, ext, ok := readImageUpload(w, r, "icon")
	if !ok {
		return
	}
	dest, err := writeImageFile(a.iconDir(), "host-thumb-"+id, ext, data, h.ThumbnailPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store thumbnail")
		return
	}
	if err := a.db.SetHostThumbnailPath(id, dest); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not store thumbnail")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"thumbnail_url": thumbnailURL("hosts", id, dest)})
}

func (a *app) handleDeleteHostThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if h.ThumbnailPath != "" {
		os.Remove(h.ThumbnailPath)
		if err := a.db.SetHostThumbnailPath(id, ""); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not update thumbnail")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"thumbnail_url": ""})
}

func (a *app) handleServeHostThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil || h.ThumbnailPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "no thumbnail")
		return
	}
	if _, ok := rbac.FromContext(r.Context()); !ok && !a.db.HostHasPublicTool(id) {
		writeError(w, http.StatusNotFound, "not_found", "no thumbnail")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, h.ThumbnailPath)
}
