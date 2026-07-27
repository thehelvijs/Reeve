package main

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type toolInput struct {
	Name             string   `json:"name"`
	Slug             string   `json:"slug"`
	Description      string   `json:"description"`
	CollectionIDs    []string `json:"collection_ids"`
	Tags             []string `json:"tags"`
	Scheme           string   `json:"scheme"`
	Address          string   `json:"address"`
	Port             int      `json:"port"`
	URL              string   `json:"url"`
	PhysicalLocation string   `json:"physical_location"`
	HostID           string   `json:"host_id"`
	SourceType       string   `json:"source_type"`
	SourceRef        string   `json:"source_ref"`
	Visibility       string   `json:"visibility"`
	LogAlertEnabled  bool     `json:"log_alert_enabled"`
}

type toolResponse struct {
	contracts.ToolDTO
	Slug            string `json:"slug"`
	SourceRef       string `json:"source_ref"`
	Visibility      string `json:"visibility"`
	CreatorID       string `json:"creator_id"`
	CanEdit         bool   `json:"can_edit"`
	LogAlertEnabled bool   `json:"log_alert_enabled"`
	IconURL         string `json:"icon_url"`
	ThumbnailURL    string `json:"thumbnail_url"`
}

// publicToolResponse is the anonymous-visible tool shape per ADR-0006: the
// catalog DTO and nothing else — no creator_id, source_ref, visibility,
// can_edit, or log_alert_enabled.
type publicToolResponse struct {
	contracts.ToolDTO
	Slug         string `json:"slug"`
	IconURL      string `json:"icon_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

func toolToResponse(t store.Tool, p auth.Principal) toolResponse {
	return toolResponse{
		ToolDTO: contracts.ToolDTO{
			ID:               t.ID,
			Name:             t.Name,
			Description:      t.Description,
			Tags:             t.Tags,
			Scheme:           t.Scheme,
			Address:          t.Address,
			Port:             t.Port,
			URL:              t.URL,
			PhysicalLocation: t.PhysicalLocation,
			HostID:           t.HostID,
			SourceType:       t.SourceType,
			Status:           contracts.StatusUnknown, // live status arrives in M6
		},
		Slug:            t.Slug,
		SourceRef:       t.SourceRef,
		Visibility:      t.Visibility,
		CreatorID:       t.CreatorID,
		CanEdit:         p.IsAdmin() || t.CreatorID == p.UserID,
		LogAlertEnabled: t.LogAlertEnabled,
		IconURL:         assetURL("tools", t.ID, "icon", t.IconPath),
		ThumbnailURL:    assetURL("tools", t.ID, "thumbnail", t.ThumbnailPath),
	}
}

// visibleCollectionRefs maps tool ids to the collections each caller may see.
// A nil principal means anonymous, which sees public collections only.
func (a *app) visibleCollectionRefs(toolIDs []string, p *auth.Principal) map[string][]contracts.CollectionRef {
	out := map[string][]contracts.CollectionRef{}
	byTool, err := a.db.CollectionsForTools(toolIDs)
	if err != nil {
		return out
	}
	for toolID, cols := range byTool {
		refs := []contracts.CollectionRef{}
		for _, c := range cols {
			visible := c.Visibility == store.VisibilityPublic
			if !visible && p != nil {
				visible, _ = a.db.CanSeeCollection(p.UserID, p.IsAdmin(), c.ID)
			}
			if !visible {
				continue
			}
			refs = append(refs, contracts.CollectionRef{
				ID: c.ID, Name: c.Name, IconURL: assetURL("collections", c.ID, "icon", c.IconPath),
			})
		}
		out[toolID] = refs
	}
	return out
}

// collectionRefsFor returns one tool's visible collections, never nil so the
// JSON is [] rather than null.
func (a *app) collectionRefsFor(toolID string, p *auth.Principal) []contracts.CollectionRef {
	refs := a.visibleCollectionRefs([]string{toolID}, p)[toolID]
	if refs == nil {
		return []contracts.CollectionRef{}
	}
	return refs
}

// applyCollectionIDs replaces a tool's collection membership, rejecting ids the
// caller cannot see.
func (a *app) applyCollectionIDs(w http.ResponseWriter, toolID string, ids []string, p auth.Principal) bool {
	for _, id := range ids {
		ok, _ := a.db.CanSeeCollection(p.UserID, p.IsAdmin(), id)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_collection", "unknown collection id")
			return false
		}
	}
	if err := a.db.SetToolCollections(toolID, ids); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not set collections")
		return false
	}
	return true
}

func (a *app) handleListTools(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	f := store.ToolFilter{
		Search:       strings.TrimSpace(r.URL.Query().Get("search")),
		CollectionID: r.URL.Query().Get("collection"),
		HostID:       r.URL.Query().Get("host"),
		SourceType:   r.URL.Query().Get("source_type"),
	}
	tools, err := a.db.ListToolsVisibleTo(p.UserID, p.IsAdmin(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list tools")
		return
	}
	hosts := a.hostMap()
	now := time.Now().UTC()
	ids := make([]string, 0, len(tools))
	for _, t := range tools {
		ids = append(ids, t.ID)
	}
	refs := a.visibleCollectionRefs(ids, &p)
	out := make([]toolResponse, 0, len(tools))
	for _, t := range tools {
		tr := toolToResponse(t, p)
		tr.Status = a.toolStatus(t, hosts, now)
		tr.Collections = refs[t.ID]
		if tr.Collections == nil {
			tr.Collections = []contracts.CollectionRef{}
		}
		out = append(out, tr)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleListPublicTools(w http.ResponseWriter, r *http.Request) {
	f := store.ToolFilter{
		Search:       strings.TrimSpace(r.URL.Query().Get("search")),
		CollectionID: r.URL.Query().Get("collection"),
		HostID:       r.URL.Query().Get("host"),
		SourceType:   r.URL.Query().Get("source_type"),
	}
	tools, err := a.db.ListPublicTools(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list tools")
		return
	}
	hosts := a.hostMap()
	now := time.Now().UTC()
	ids := make([]string, 0, len(tools))
	for _, t := range tools {
		ids = append(ids, t.ID)
	}
	refs := a.visibleCollectionRefs(ids, nil)
	out := make([]publicToolResponse, 0, len(tools))
	for _, t := range tools {
		cols := refs[t.ID]
		if cols == nil {
			cols = []contracts.CollectionRef{}
		}
		out = append(out, publicToolResponse{
			ToolDTO: contracts.ToolDTO{
				ID:               t.ID,
				Name:             t.Name,
				Description:      t.Description,
				Collections:      cols,
				Tags:             t.Tags,
				Scheme:           t.Scheme,
				Address:          t.Address,
				Port:             t.Port,
				URL:              t.URL,
				PhysicalLocation: t.PhysicalLocation,
				HostID:           t.HostID,
				SourceType:       t.SourceType,
				Status:           a.toolStatus(t, hosts, now),
			},
			Slug:         t.Slug,
			IconURL:      assetURL("tools", t.ID, "icon", t.IconPath),
			ThumbnailURL: assetURL("tools", t.ID, "thumbnail", t.ThumbnailPath),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreateTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	var in toolInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "tool name is required")
		return
	}
	if !validSourceType(in.SourceType) {
		writeError(w, http.StatusBadRequest, "invalid_source_type", "source_type must be manual, systemd, docker, or cron")
		return
	}
	if !validVisibility(in.Visibility) {
		writeError(w, http.StatusBadRequest, "invalid_visibility", "visibility must be public or restricted")
		return
	}
	if !checkEndpointFields(w, in) {
		return
	}
	slug, ok := a.resolveSlug(w, in.Slug, in.Name, "")
	if !ok {
		return
	}
	t, err := a.db.CreateTool(store.Tool{
		Name: in.Name, Slug: slug, Description: in.Description, Tags: in.Tags,
		Scheme: in.Scheme, Address: in.Address, Port: in.Port, URL: in.URL,
		PhysicalLocation: in.PhysicalLocation, HostID: in.HostID, SourceType: in.SourceType,
		SourceRef: in.SourceRef, Visibility: in.Visibility, CreatorID: p.UserID,
		LogAlertEnabled: in.LogAlertEnabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create tool")
		return
	}
	if !a.applyCollectionIDs(w, t.ID, in.CollectionIDs, p) {
		a.db.DeleteTool(t.ID)
		return
	}
	tr := toolToResponse(t, p)
	tr.Collections = a.collectionRefsFor(t.ID, &p)
	writeJSON(w, http.StatusCreated, tr)
}

func (a *app) handleGetTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return
	}
	tr := toolToResponse(t, p)
	tr.Status = a.toolStatus(t, a.hostMap(), time.Now().UTC())
	tr.Collections = a.collectionRefsFor(t.ID, &p)
	writeJSON(w, http.StatusOK, tr)
}

func (a *app) handleUpdateTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	var in toolInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "tool name is required")
		return
	}
	if !validSourceType(in.SourceType) || !validVisibility(in.Visibility) {
		writeError(w, http.StatusBadRequest, "invalid_field", "invalid source_type or visibility")
		return
	}
	if !checkEndpointFields(w, in) {
		return
	}
	// A rename leaves the slug alone: /go/ URLs are bookmarked, and silently
	// repointing one is worse than a slug that no longer matches the name.
	if in.Slug != "" && in.Slug != t.Slug {
		slug, ok := a.resolveSlug(w, in.Slug, in.Name, t.ID)
		if !ok {
			return
		}
		t.Slug = slug
	}
	t.Name, t.Description, t.Tags = in.Name, in.Description, in.Tags
	t.Scheme, t.Address, t.Port, t.URL = in.Scheme, in.Address, in.Port, in.URL
	t.PhysicalLocation, t.HostID, t.SourceRef = in.PhysicalLocation, in.HostID, in.SourceRef
	t.LogAlertEnabled = in.LogAlertEnabled
	// Empty enum fields keep the existing value rather than blanking it.
	if in.SourceType != "" {
		t.SourceType = in.SourceType
	}
	if in.Visibility != "" {
		t.Visibility = in.Visibility
	}
	if err := a.db.UpdateTool(t); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not update tool")
		return
	}
	if !a.applyCollectionIDs(w, t.ID, in.CollectionIDs, p) {
		return
	}
	tr := toolToResponse(t, p)
	tr.Collections = a.collectionRefsFor(t.ID, &p)
	writeJSON(w, http.StatusOK, tr)
}

func (a *app) handleDeleteTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadEditableTool(w, r, p)
	if !ok {
		return
	}
	if err := a.db.DeleteTool(t.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not delete tool")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleToolEvents(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return
	}
	events, err := a.db.ListAlertEventsForTool(t.ID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list events")
		return
	}
	if events == nil {
		events = []store.AlertEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

// loadVisibleTool fetches the tool and 404s if the principal may not see it.
func (a *app) loadVisibleTool(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Tool, bool) {
	id := r.PathValue("id")
	ok, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return store.Tool{}, false
	}
	t, err := a.db.GetTool(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "tool not found")
		return store.Tool{}, false
	}
	return t, true
}

// loadEditableTool additionally requires the principal to be the creator or an
// admin: 404 if not visible, 403 if visible but not owned.
func (a *app) loadEditableTool(w http.ResponseWriter, r *http.Request, p auth.Principal) (store.Tool, bool) {
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return store.Tool{}, false
	}
	if !p.IsAdmin() && t.CreatorID != p.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "only the creator or an admin can modify this tool")
		return store.Tool{}, false
	}
	return t, true
}

// resolveSlug settles a tool's /go/ name. An empty request slug is derived
// from the tool name and made unique automatically, because most people never
// think about it. One typed by hand is taken literally: it is a URL someone is
// about to share, so a collision is a 409 to act on rather than a silent
// rename to something they did not choose.
func (a *app) resolveSlug(w http.ResponseWriter, requested, name, excludeID string) (string, bool) {
	if requested == "" {
		slug, err := a.db.UniqueToolSlug(store.Slugify(name), excludeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not derive a URL name")
			return "", false
		}
		return slug, true
	}
	slug := store.Slugify(requested)
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid_slug",
			"the URL name must contain a letter or a digit")
		return "", false
	}
	free, err := a.db.UniqueToolSlug(slug, excludeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not check the URL name")
		return "", false
	}
	if free != slug {
		writeError(w, http.StatusConflict, "slug_taken", "another tool already uses /go/"+slug)
		return "", false
	}
	return slug, true
}

func validSourceType(s string) bool {
	switch s {
	case "", "manual", "systemd", "docker", "cron":
		return true
	}
	return false
}

func validVisibility(s string) bool {
	return s == "" || s == store.VisibilityPublic || s == store.VisibilityRestricted
}

// validScheme allows the two the tool form offers. Empty means "unset", which
// resolveEndpoint turns into http. Anything else ends up in a Location header
// and in the endpoint JSON, and javascript: and data: do not belong in either.
func validScheme(s string) bool {
	return s == "" || s == "http" || s == "https"
}

// validToolURL accepts an absolute http or https URL, or none at all. A tool's
// URL is handed straight to a redirect, so it is a navigation target typed by a
// user and has to be checked like one.
func validToolURL(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// checkEndpointFields writes the error response itself, so both the create and
// the update path reject the same values with the same message.
func checkEndpointFields(w http.ResponseWriter, in toolInput) bool {
	if !validScheme(in.Scheme) {
		writeError(w, http.StatusBadRequest, "invalid_scheme", "scheme must be http or https")
		return false
	}
	if !validToolURL(in.URL) {
		writeError(w, http.StatusBadRequest, "invalid_url", "url must be an absolute http or https URL")
		return false
	}
	return true
}

// toolStatus derives a tool's live status from agent telemetry; tools without an
// agent signal report "unknown".
func (a *app) toolStatus(t store.Tool, hosts map[string]store.Host, now time.Time) contracts.ToolStatus {
	return a.agentToolStatus(t, hosts, now)
}

// agentToolStatus derives status from the tool's agent source and host
// telemetry. Tools without a monitored source report "unknown".
func (a *app) agentToolStatus(t store.Tool, hosts map[string]store.Host, now time.Time) contracts.ToolStatus {
	if !agentMonitored(t) {
		return contracts.StatusUnknown
	}
	host, ok := hosts[t.HostID]
	if !ok {
		return contracts.StatusUnknown
	}
	if !host.Online(now) {
		return contracts.StatusAgentOffline
	}
	switch t.SourceType {
	case "systemd":
		active, ok := a.db.LookupServiceState(t.HostID, t.SourceRef)
		if !ok {
			return contracts.StatusUnknown
		}
		if active == "active" {
			return contracts.StatusUp
		}
		return contracts.StatusDown
	case "docker":
		state, health, ok := a.db.LookupContainerState(t.HostID, t.SourceRef)
		if !ok {
			return contracts.StatusUnknown
		}
		if state == "running" && (health == "" || health == "healthy") {
			return contracts.StatusUp
		}
		return contracts.StatusDown
	case "cron":
		if a.db.LookupCronExists(t.HostID, t.SourceRef) {
			return contracts.StatusUp
		}
		return contracts.StatusUnknown
	}
	return contracts.StatusUnknown
}

// agentMonitored reports whether a tool derives status from an agent source on
// a host (systemd/docker/cron).
func agentMonitored(t store.Tool) bool {
	return t.HostID != "" && t.SourceType != "manual" && t.SourceRef != ""
}

// hostMap loads all hosts keyed by id for status derivation.
func (a *app) hostMap() map[string]store.Host {
	hosts, err := a.db.ListHosts()
	if err != nil {
		return map[string]store.Host{}
	}
	m := make(map[string]store.Host, len(hosts))
	for _, h := range hosts {
		m[h.ID] = h
	}
	return m
}
