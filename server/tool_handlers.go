package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type toolInput struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
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
	SourceRef       string   `json:"source_ref"`
	Visibility      string   `json:"visibility"`
	CreatorID       string   `json:"creator_id"`
	CanEdit         bool     `json:"can_edit"`
	LogAlertEnabled bool     `json:"log_alert_enabled"`
	Tags            []string `json:"tags"`
	IconURL         string   `json:"icon_url"`
	ThumbnailURL    string   `json:"thumbnail_url"`
}

// publicToolResponse is the anonymous-visible tool shape per ADR-0006: no
// creator_id, source_ref, visibility, can_edit, or log_alert_enabled.
type publicToolResponse struct {
	ID               string                    `json:"id"`
	Name             string                    `json:"name"`
	Description      string                    `json:"description"`
	Collections      []contracts.CollectionRef `json:"collections"`
	Tags             []string                  `json:"tags"`
	Scheme           string                    `json:"scheme"`
	Address          string                    `json:"address"`
	Port             int                       `json:"port,omitempty"`
	URL              string                    `json:"url,omitempty"`
	PhysicalLocation string                    `json:"physical_location,omitempty"`
	HostID           string                    `json:"host_id,omitempty"`
	SourceType       string                    `json:"source_type"`
	Status           contracts.ToolStatus      `json:"status"`
	IconURL          string                    `json:"icon_url"`
	ThumbnailURL     string                    `json:"thumbnail_url"`
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
		SourceRef:       t.SourceRef,
		Visibility:      t.Visibility,
		CreatorID:       t.CreatorID,
		CanEdit:         p.IsAdmin() || t.CreatorID == p.UserID,
		LogAlertEnabled: t.LogAlertEnabled,
		Tags:            t.Tags,
		IconURL:         iconURL("tools", t.ID, t.IconPath),
		ThumbnailURL:    thumbnailURL("tools", t.ID, t.ThumbnailPath),
	}
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
	out := make([]toolResponse, 0, len(tools))
	for _, t := range tools {
		tr := toolToResponse(t, p)
		tr.Status = a.toolStatus(t, hosts, now)
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
	out := make([]publicToolResponse, 0, len(tools))
	for _, t := range tools {
		out = append(out, publicToolResponse{
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
			Status:           a.toolStatus(t, hosts, now),
			IconURL:          iconURL("tools", t.ID, t.IconPath),
			ThumbnailURL:     thumbnailURL("tools", t.ID, t.ThumbnailPath),
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
	t, err := a.db.CreateTool(store.Tool{
		Name: in.Name, Description: in.Description, Tags: in.Tags,
		Scheme: in.Scheme, Address: in.Address, Port: in.Port, URL: in.URL,
		PhysicalLocation: in.PhysicalLocation, HostID: in.HostID, SourceType: in.SourceType,
		SourceRef: in.SourceRef, Visibility: in.Visibility, CreatorID: p.UserID,
		LogAlertEnabled: in.LogAlertEnabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create tool")
		return
	}
	writeJSON(w, http.StatusCreated, toolToResponse(t, p))
}

func (a *app) handleGetTool(w http.ResponseWriter, r *http.Request) {
	p, _ := rbac.FromContext(r.Context())
	t, ok := a.loadVisibleTool(w, r, p)
	if !ok {
		return
	}
	tr := toolToResponse(t, p)
	tr.Status = a.toolStatus(t, a.hostMap(), time.Now().UTC())
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
	writeJSON(w, http.StatusOK, toolToResponse(t, p))
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
