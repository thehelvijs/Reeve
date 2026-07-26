package main

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// Where a resolved address came from, so a caller can tell a pinned tool from
// one that follows its host.
const (
	endpointSourceURL     = "url"
	endpointSourceAddress = "address"
	endpointSourceHost    = "host"
)

// errNoEndpoint means the tool names no place to go: no URL, no address, and
// either no host or a host that has never reported one.
var errNoEndpoint = errors.New(
	"this tool has no URL or address, and its host has not reported one yet")

type endpointView struct {
	ToolID     string `json:"tool_id"`
	Slug       string `json:"slug"`
	URL        string `json:"url"`
	Scheme     string `json:"scheme"`
	Address    string `json:"address"`
	Port       int    `json:"port,omitempty"`
	HostID     string `json:"host_id,omitempty"`
	HostIP     string `json:"host_ip,omitempty"`
	Source     string `json:"source"`
	HostOnline bool   `json:"host_online"`
}

// resolveEndpoint answers where a tool is right now. A URL typed on the tool
// wins outright, then an address typed on the tool, and only a tool that gave
// neither falls back to its host's last reported address. That ordering is what
// makes "leave the address blank" the way to follow a host, without changing
// what an already-configured tool does.
func resolveEndpoint(t store.Tool, host store.Host) (endpointView, error) {
	v := endpointView{ToolID: t.ID, Slug: t.Slug, Scheme: t.Scheme, Port: t.Port}
	if t.HostID != "" {
		v.HostID = t.HostID
		v.HostIP = host.IPAddress
	}

	if t.URL != "" {
		v.URL = t.URL
		v.Source = endpointSourceURL
		return v, nil
	}

	switch {
	case t.Address != "":
		v.Address = t.Address
		v.Source = endpointSourceAddress
	case host.IPAddress != "":
		v.Address = host.IPAddress
		v.Source = endpointSourceHost
	default:
		return endpointView{}, errNoEndpoint
	}

	if v.Scheme == "" {
		v.Scheme = "http"
	}
	authority := v.Address
	if v.Port != 0 {
		authority = net.JoinHostPort(v.Address, strconv.Itoa(v.Port))
	}
	v.URL = v.Scheme + "://" + authority
	return v, nil
}

// loadEndpoint resolves the tool a /go/ or /endpoints/ path names, writing the
// error response itself. A tool the caller may not see is indistinguishable
// from one that does not exist, so neither can be used to enumerate the
// catalog.
func (a *app) loadEndpoint(w http.ResponseWriter, r *http.Request) (endpointView, bool) {
	t, err := a.db.GetToolBySlug(r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "no tool with that name")
		return endpointView{}, false
	}
	if !a.canSeeToolForEndpoint(r, t) {
		writeError(w, http.StatusNotFound, "not_found", "no tool with that name")
		return endpointView{}, false
	}

	var host store.Host
	if t.HostID != "" {
		if h, err := a.db.GetHost(t.HostID); err == nil {
			host = h
		}
	}
	v, err := resolveEndpoint(t, host)
	if err != nil {
		writeError(w, http.StatusConflict, "no_endpoint", err.Error())
		return endpointView{}, false
	}
	v.HostOnline = host.Online(time.Now().UTC())
	return v, true
}

// canSeeToolForEndpoint applies the same visibility ladder as the icon routes:
// a public tool is anonymous-visible, anything else needs a principal who may
// see it.
func (a *app) canSeeToolForEndpoint(r *http.Request, t store.Tool) bool {
	if t.Visibility == store.VisibilityPublic {
		return true
	}
	p, ok := rbac.FromContext(r.Context())
	if !ok {
		return false
	}
	seen, _ := a.db.CanSeeTool(p.UserID, p.IsAdmin(), t.ID)
	return seen
}

// handleGoToTool redirects to wherever the tool is now. This is the URL worth
// bookmarking: it survives the host's DHCP lease changing underneath it.
func (a *app) handleGoToTool(w http.ResponseWriter, r *http.Request) {
	v, ok := a.loadEndpoint(w, r)
	if !ok {
		return
	}
	// A stale address still beats no answer, so an offline host redirects to
	// the last one it reported.
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, v.URL, http.StatusFound)
}

// handleGetEndpoint answers the same question as JSON, for scripts and for the
// UI's copyable endpoint.
func (a *app) handleGetEndpoint(w http.ResponseWriter, r *http.Request) {
	v, ok := a.loadEndpoint(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, v)
}
