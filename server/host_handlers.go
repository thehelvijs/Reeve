package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

type hostView struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	OS               string           `json:"os"`
	PhysicalLocation string           `json:"physical_location"`
	IPAddress        string           `json:"ip_address"`
	AgentVersion     string           `json:"agent_version"`
	AutoUpdate       string           `json:"auto_update"`
	UpdateState      string           `json:"update_state"`
	Status           string           `json:"status"` // online | offline | never
	LastSeenAt       string           `json:"last_seen_at,omitempty"`
	Metrics          *hostMetricsView `json:"metrics,omitempty"`
	IconURL          string           `json:"icon_url"`
	ThumbnailURL     string           `json:"thumbnail_url"`
	Latitude         *float64         `json:"latitude,omitempty"`
	Longitude        *float64         `json:"longitude,omitempty"`
}

// hostMetricsView is the host's most recent sample, for at-a-glance load on the
// dashboard. Absent when the host has never pushed.
type hostMetricsView struct {
	CPUPct    float64 `json:"cpu_pct"`
	MemUsed   uint64  `json:"mem_used"`
	MemTotal  uint64  `json:"mem_total"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskTotal uint64  `json:"disk_total"`
	At        string  `json:"at"`
}

func hostToView(h store.Host, now time.Time, uc updateContext) hostView {
	last := ""
	if h.LastSeenAt != nil {
		last = h.LastSeenAt.Format(time.RFC3339)
	}
	return hostView{
		ID: h.ID, Name: h.Name, OS: h.OS, PhysicalLocation: h.PhysicalLocation,
		AgentVersion: h.AgentVersion, AutoUpdate: h.AutoUpdate,
		IPAddress:   h.IPAddress,
		UpdateState: updateStateFor(h, uc, now),
		Status:      hostStatus(h, now), LastSeenAt: last,
		IconURL: assetURL("hosts", h.ID, "icon", h.IconPath), ThumbnailURL: assetURL("hosts", h.ID, "thumbnail", h.ThumbnailPath),
		Latitude: h.Latitude, Longitude: h.Longitude,
	}
}

// hostStatus reports online/offline/never from the host's last-seen heartbeat.
func hostStatus(h store.Host, now time.Time) string {
	if h.LastSeenAt == nil {
		return "never"
	}
	if h.Online(now) {
		return "online"
	}
	return "offline"
}

// publicHostView is the anonymous-visible host shape per ADR-0006: no os,
// physical_location, agent_version, or last_seen_at.
type publicHostView struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	IconURL      string   `json:"icon_url"`
	ThumbnailURL string   `json:"thumbnail_url"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
}

func (a *app) handleListHosts(w http.ResponseWriter, _ *http.Request) {
	hosts, err := a.db.ListHosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list hosts")
		return
	}
	latest, err := a.db.LatestHostMetrics()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read metrics")
		return
	}
	now := time.Now().UTC()
	uc := a.updateContext()
	out := make([]hostView, 0, len(hosts))
	for _, h := range hosts {
		v := hostToView(h, now, uc)
		if m, ok := latest[h.ID]; ok {
			v.Metrics = &hostMetricsView{
				CPUPct: m.CPUPct, MemUsed: m.MemUsed, MemTotal: m.MemTotal,
				DiskUsed: m.DiskUsed, DiskTotal: m.DiskTotal, At: m.TS.Format(time.RFC3339),
			}
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleListPublicHosts(w http.ResponseWriter, _ *http.Request) {
	hosts, err := a.db.ListPublicHosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list hosts")
		return
	}
	now := time.Now().UTC()
	out := make([]publicHostView, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, publicHostView{ID: h.ID, Name: h.Name, Status: hostStatus(h, now), IconURL: assetURL("hosts", h.ID, "icon", h.IconPath), ThumbnailURL: assetURL("hosts", h.ID, "thumbnail", h.ThumbnailPath), Latitude: h.Latitude, Longitude: h.Longitude})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCreateHost(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name             string `json:"name"`
		OS               string `json:"os"`
		PhysicalLocation string `json:"physical_location"`
		OfflineAfterSecs int    `json:"offline_after_secs"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "host name is required")
		return
	}
	if in.OS == "" {
		in.OS = "linux"
	}
	token, hash := auth.NewAgentToken()
	h, err := a.db.CreateHost(in.Name, in.OS, in.PhysicalLocation, hash, in.OfflineAfterSecs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create host")
		return
	}
	// The enrollment token is returned exactly once.
	writeJSON(w, http.StatusCreated, map[string]any{
		"host":            hostToView(h, time.Now().UTC(), a.updateContext()),
		"enroll_token":    token,
		"install_command": a.agentInstallCommand(token),
		"run_command":     a.agentRunCommand(token),
	})
}

func (a *app) handleClearHostMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if err := a.db.DeleteHostMetrics(id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not clear metrics")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleDeleteHost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.db.DeleteHost(id); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "host not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not delete host")
		return
	}
	a.db.DeleteHostThresholds(id)
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateHostLocation sets a host's physical location and optional map
// coordinates. Admin only.
func (a *app) handleUpdateHostLocation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		PhysicalLocation string   `json:"physical_location"`
		Latitude         *float64 `json:"latitude"`
		Longitude        *float64 `json:"longitude"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := a.db.UpdateHostLocation(id, in.PhysicalLocation, in.Latitude, in.Longitude); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "host not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not update host")
		return
	}
	updated, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load host")
		return
	}
	writeJSON(w, http.StatusOK, hostToView(updated, time.Now().UTC(), a.updateContext()))
}

func (a *app) handleHostEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	events, err := a.db.ListAlertEventsForHost(id, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not list events")
		return
	}
	if events == nil {
		events = []store.AlertEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

type inventoryResponse struct {
	Services   []inventoryItem `json:"services"`
	Containers []inventoryItem `json:"containers"`
	CronJobs   []inventoryItem `json:"cron_jobs"`
}

type inventoryItem struct {
	SourceType string `json:"source_type"`
	SourceRef  string `json:"source_ref"`
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	Linked     bool   `json:"linked"`
}

func (a *app) handleHostInventory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	linked, _ := a.db.SourceRefsForHost(id)
	services, _ := a.db.ListServiceStatus(id)
	containers, _ := a.db.ListContainerStatus(id)
	crons, _ := a.db.ListCronJobs(id)

	resp := inventoryResponse{Services: []inventoryItem{}, Containers: []inventoryItem{}, CronJobs: []inventoryItem{}}
	for _, s := range services {
		resp.Services = append(resp.Services, inventoryItem{
			SourceType: "systemd", SourceRef: s.Unit, Name: s.Unit,
			Detail: s.ActiveState + "/" + s.SubState, Linked: linked["systemd:"+s.Unit],
		})
	}
	for _, c := range containers {
		resp.Containers = append(resp.Containers, inventoryItem{
			SourceType: "docker", SourceRef: c.ContainerID, Name: c.Name,
			Detail: c.Image + " · " + c.State, Linked: linked["docker:"+c.ContainerID],
		})
	}
	for _, c := range crons {
		resp.CronJobs = append(resp.CronJobs, inventoryItem{
			SourceType: "cron", SourceRef: c.Name, Name: c.Name,
			Detail: c.Schedule, Linked: linked["cron:"+c.Name],
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// agentInstallCommand renders the one-line host installer (systemd, full
// visibility). Mirrors agentRunCommand's URL resolution.
func (a *app) agentInstallCommand(token string) string {
	url := a.cfg.PublicURL
	if url == "" {
		url = "http://" + a.cfg.Addr
	}
	url = strings.TrimSuffix(url, "/")
	return fmt.Sprintf(
		"curl -fsSL %s/install.sh | sudo "+
			"REEVE_SERVER_URL=%s REEVE_AGENT_TOKEN=%s bash",
		url, url, token)
}

// agentRunCommand renders a copy-paste Docker command to enroll the agent.
func (a *app) agentRunCommand(token string) string {
	url := a.cfg.PublicURL
	if url == "" {
		url = "http://" + a.cfg.Addr
	}
	return fmt.Sprintf(
		"docker run -d --name reeve-agent --restart unless-stopped "+
			"-v /var/run/docker.sock:/var/run/docker.sock:ro "+
			"-e REEVE_SERVER_URL=%s -e REEVE_AGENT_TOKEN=%s "+
			"ghcr.io/thehelvijs/reeve-agent:latest",
		url, token)
}
