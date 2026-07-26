package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/thehelvijs/Reeve/server/internal/auth"
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

// imageTarget is what an entity tells its image slot: where the stored file is,
// whether anonymous callers may see it, and who owns it.
type imageTarget struct {
	path      string
	public    bool
	creatorID string
}

// imageSlot is one uploadable image on one entity type. Every slot differs only
// in these fields, so upload, delete and serve are written once.
type imageSlot struct {
	kind   string // URL segment: tools, hosts, collections
	leaf   string // icon or thumbnail
	entity string // noun for the not-found message
	prefix string // stored filename prefix
	// load reports the entity's current state, or an error when it is absent.
	load func(*app, string) (imageTarget, error)
	// setPath records the stored file, or clears it when dest is empty.
	setPath func(*app, string, string) error
	// canEdit gates upload and delete beyond what the route already gates.
	canEdit func(*app, auth.Principal, string, imageTarget) bool
	// canSee gates serving a non-public entity's file.
	canSee func(*app, *http.Request, string) bool
}

// toolCanEdit: a tool's images are editable by its creator and by any admin.
func toolCanEdit(_ *app, p auth.Principal, _ string, tgt imageTarget) bool {
	return p.IsAdmin() || tgt.creatorID == p.UserID
}

// toolCanSee: a restricted tool's file needs a principal it is shared with.
func toolCanSee(a *app, r *http.Request, id string) bool {
	p, ok := rbac.FromContext(r.Context())
	if !ok {
		return false
	}
	seen, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), id)
	return seen
}

// hostCanSee: any authenticated caller, and an anonymous one only when the host
// backs a public tool.
func hostCanSee(a *app, r *http.Request, id string) bool {
	if _, ok := rbac.FromContext(r.Context()); ok {
		return true
	}
	return a.db.HostHasPublicTool(id)
}

// alwaysEditable is for slots whose route is already admin-gated.
func alwaysEditable(*app, auth.Principal, string, imageTarget) bool { return true }

func toolTarget(a *app, id string, thumbnail bool) (imageTarget, error) {
	t, err := a.db.GetTool(id)
	if err != nil {
		return imageTarget{}, err
	}
	path := t.IconPath
	if thumbnail {
		path = t.ThumbnailPath
	}
	return imageTarget{path: path, public: t.Visibility == store.VisibilityPublic, creatorID: t.CreatorID}, nil
}

func hostTarget(a *app, id string, thumbnail bool) (imageTarget, error) {
	h, err := a.db.GetHost(id)
	if err != nil {
		return imageTarget{}, err
	}
	path := h.IconPath
	if thumbnail {
		path = h.ThumbnailPath
	}
	return imageTarget{path: path}, nil
}

var toolIcon = imageSlot{
	kind: "tools", leaf: "icon", entity: "tool", prefix: "tool-",
	load:    func(a *app, id string) (imageTarget, error) { return toolTarget(a, id, false) },
	setPath: func(a *app, id, dest string) error { return a.db.SetToolIconPath(id, dest) },
	canEdit: toolCanEdit,
	canSee:  toolCanSee,
}

var toolThumbnail = imageSlot{
	kind: "tools", leaf: "thumbnail", entity: "tool", prefix: "tool-thumb-",
	load:    func(a *app, id string) (imageTarget, error) { return toolTarget(a, id, true) },
	setPath: func(a *app, id, dest string) error { return a.db.SetToolThumbnailPath(id, dest) },
	canEdit: toolCanEdit,
	canSee:  toolCanSee,
}

var hostIcon = imageSlot{
	kind: "hosts", leaf: "icon", entity: "host", prefix: "host-",
	load:    func(a *app, id string) (imageTarget, error) { return hostTarget(a, id, false) },
	setPath: func(a *app, id, dest string) error { return a.db.SetHostIconPath(id, dest) },
	canEdit: alwaysEditable,
	canSee:  hostCanSee,
}

var hostThumbnail = imageSlot{
	kind: "hosts", leaf: "thumbnail", entity: "host", prefix: "host-thumb-",
	load:    func(a *app, id string) (imageTarget, error) { return hostTarget(a, id, true) },
	setPath: func(a *app, id, dest string) error { return a.db.SetHostThumbnailPath(id, dest) },
	canEdit: alwaysEditable,
	canSee:  hostCanSee,
}

var collectionIcon = imageSlot{
	kind: "collections", leaf: "icon", entity: "collection", prefix: "collection-",
	load: func(a *app, id string) (imageTarget, error) {
		c, err := a.db.GetCollection(id)
		if err != nil {
			return imageTarget{}, err
		}
		return imageTarget{path: c.IconPath, public: c.Visibility == store.VisibilityPublic}, nil
	},
	setPath: func(a *app, id, dest string) error { return a.db.SetCollectionIconPath(id, dest) },
	canEdit: func(a *app, p auth.Principal, id string, _ imageTarget) bool {
		can, _ := a.db.CanEditCollection(p.UserID, p.IsAdmin(), id)
		return can
	},
	canSee: func(a *app, r *http.Request, id string) bool {
		p, ok := rbac.FromContext(r.Context())
		if !ok {
			return false
		}
		seen, _ := a.db.CanSeeCollection(p.UserID, p.IsAdmin(), id)
		return seen
	},
}

// loadWritable resolves the entity a write names and authorizes the caller,
// answering 404 or 403 itself.
func (a *app) loadWritable(w http.ResponseWriter, r *http.Request, s imageSlot, id string) (imageTarget, bool) {
	tgt, err := s.load(a, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", s.entity+" not found")
		return imageTarget{}, false
	}
	p, _ := rbac.FromContext(r.Context())
	if !s.canEdit(a, p, id, tgt) {
		writeError(w, http.StatusForbidden, "forbidden", "not allowed to edit this "+s.entity)
		return imageTarget{}, false
	}
	return tgt, true
}

// uploadImage stores an uploaded image in a slot, replacing whatever was there.
func (a *app) uploadImage(s imageSlot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		tgt, ok := a.loadWritable(w, r, s, id)
		if !ok {
			return
		}
		data, ext, ok := readImageUpload(w, r, "icon")
		if !ok {
			return
		}
		dest, err := writeImageFile(a.iconDir(), s.prefix+id, ext, data, tgt.path)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not store "+s.leaf)
			return
		}
		if err := s.setPath(a, id, dest); err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not store "+s.leaf)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{s.leaf + "_url": assetURL(s.kind, id, s.leaf, dest)})
	}
}

// deleteImage clears a slot, removing the file on disk.
func (a *app) deleteImage(s imageSlot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		tgt, ok := a.loadWritable(w, r, s, id)
		if !ok {
			return
		}
		if tgt.path != "" {
			os.Remove(tgt.path)
			if err := s.setPath(a, id, ""); err != nil {
				writeError(w, http.StatusInternalServerError, "internal", "could not update "+s.leaf)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{s.leaf + "_url": ""})
	}
}

// serveImage serves a slot's file, respecting visibility: a public entity's file
// goes to anyone, anything else only to a principal who may see it. A file the
// caller may not read is indistinguishable from one that does not exist.
func (a *app) serveImage(s imageSlot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		tgt, err := s.load(a, id)
		if err != nil || tgt.path == "" {
			writeError(w, http.StatusNotFound, "not_found", "no "+s.leaf)
			return
		}
		if !tgt.public && !s.canSee(a, r, id) {
			writeError(w, http.StatusNotFound, "not_found", "no "+s.leaf)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, tgt.path)
	}
}
