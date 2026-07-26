package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// The resolution order is the whole feature: a typed address must keep working
// exactly as before, and only a tool that gave none follows its host.
func TestResolveEndpoint(t *testing.T) {
	cases := []struct {
		name       string
		tool       store.Tool
		host       store.Host
		wantURL    string
		wantSource string
		wantErr    bool
	}{
		{
			name:       "explicit url wins over everything",
			tool:       store.Tool{URL: "https://grafana.example.com/d/abc", Address: "10.0.0.1", HostID: "h1"},
			host:       store.Host{IPAddress: "192.168.1.42"},
			wantURL:    "https://grafana.example.com/d/abc",
			wantSource: "url",
		},
		{
			name:       "typed address beats the host ip",
			tool:       store.Tool{Scheme: "http", Address: "grafana.lan", Port: 3000, HostID: "h1"},
			host:       store.Host{IPAddress: "192.168.1.42"},
			wantURL:    "http://grafana.lan:3000",
			wantSource: "address",
		},
		{
			name:       "blank address follows the host",
			tool:       store.Tool{Scheme: "http", Port: 3000, HostID: "h1"},
			host:       store.Host{IPAddress: "192.168.1.42"},
			wantURL:    "http://192.168.1.42:3000",
			wantSource: "host",
		},
		{
			name:       "scheme defaults to http",
			tool:       store.Tool{Port: 8080, HostID: "h1"},
			host:       store.Host{IPAddress: "192.168.1.42"},
			wantURL:    "http://192.168.1.42:8080",
			wantSource: "host",
		},
		{
			name:       "no port leaves the authority bare",
			tool:       store.Tool{Scheme: "https", HostID: "h1"},
			host:       store.Host{IPAddress: "192.168.1.42"},
			wantURL:    "https://192.168.1.42",
			wantSource: "host",
		},
		{
			name:    "host that has never reported has no answer",
			tool:    store.Tool{Scheme: "http", Port: 3000, HostID: "h1"},
			host:    store.Host{},
			wantErr: true,
		},
		{
			name:    "no url, no address, no host",
			tool:    store.Tool{Scheme: "http", Port: 3000},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveEndpoint(tc.tool, tc.host)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveEndpoint = %+v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveEndpoint: %v", err)
			}
			if got.URL != tc.wantURL {
				t.Errorf("url = %q, want %q", got.URL, tc.wantURL)
			}
			if got.Source != tc.wantSource {
				t.Errorf("source = %q, want %q", got.Source, tc.wantSource)
			}
		})
	}
}

// An IPv6 address has to come back bracketed or the URL is unusable.
func TestResolveEndpointBracketsIPv6(t *testing.T) {
	got, err := resolveEndpoint(
		store.Tool{Scheme: "http", Port: 3000, HostID: "h1"},
		store.Host{IPAddress: "fd00::42"})
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if got.URL != "http://[fd00::42]:3000" {
		t.Errorf("url = %q, want the address bracketed", got.URL)
	}
}

// noRedirect returns a client sharing c's cookies but surfacing a 302 instead
// of chasing it, since the targets here are addresses nothing is listening on.
func noRedirect(c *http.Client) *http.Client {
	var jar http.CookieJar
	if c != nil {
		jar = c.Jar
	}
	return &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// pushIP enrolls a host, reports an address for it, and returns the host id.
func pushIP(t *testing.T, ts *testServer, admin *http.Client, name, ip string) string {
	t.Helper()
	hostID, token := enrollHost(t, ts, admin, name)
	p := samplePush()
	p.IPAddress = ip
	resp, data := ts.do(t, nil, http.MethodPost, "/api/v1/ingest", p,
		map[string]string{"Authorization": "Bearer " + token})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ingest = %d: %s", resp.StatusCode, data)
	}
	return hostID
}

// The point of the whole feature: the redirect follows the host's current
// address, and follows it again when the lease changes.
func TestGoRedirectFollowsTheHostAddress(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID := pushIP(t, ts, admin, "pc2", "192.168.1.42")

	tool := createTool(t, ts, admin, toolInput{
		Name: "Grafana", SourceType: "manual", Visibility: "public",
		HostID: hostID, Scheme: "http", Port: 3000,
	})
	if tool.Slug != "grafana" {
		t.Fatalf("slug = %q, want grafana", tool.Slug)
	}

	resp, _ := ts.do(t, noRedirect(nil), http.MethodGet, "/go/grafana", nil, nil)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "http://192.168.1.42:3000" {
		t.Fatalf("Location = %q, want http://192.168.1.42:3000", got)
	}

	// The lease changes; the same URL must now point at the new address.
	p := samplePush()
	p.IPAddress = "192.168.1.99"
	if err := ts.app.db.ApplyPush(hostID, p, time.Now().UTC()); err != nil {
		t.Fatalf("ApplyPush: %v", err)
	}

	resp, _ = ts.do(t, noRedirect(nil), http.MethodGet, "/go/grafana", nil, nil)
	if got := resp.Header.Get("Location"); got != "http://192.168.1.99:3000" {
		t.Errorf("Location after the lease change = %q, want http://192.168.1.99:3000", got)
	}
}

// A push that could not work out its address must not wipe the last known one.
func TestEmptyReportedAddressKeepsTheLastKnownOne(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID := pushIP(t, ts, admin, "pc2", "192.168.1.42")

	blank := samplePush()
	blank.IPAddress = ""
	if err := ts.app.db.ApplyPush(hostID, blank, time.Now().UTC()); err != nil {
		t.Fatalf("ApplyPush: %v", err)
	}
	h, err := ts.app.db.GetHost(hostID)
	if err != nil {
		t.Fatalf("GetHost: %v", err)
	}
	if h.IPAddress != "192.168.1.42" {
		t.Errorf("ip_address = %q, want the previous 192.168.1.42", h.IPAddress)
	}
}

func TestEndpointJSON(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID := pushIP(t, ts, admin, "pc2", "192.168.1.42")
	createTool(t, ts, admin, toolInput{
		Name: "Grafana", SourceType: "manual", Visibility: "public",
		HostID: hostID, Scheme: "http", Port: 3000,
	})

	resp, data := ts.do(t, nil, http.MethodGet, "/api/v1/endpoints/grafana", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, data)
	}
	var v endpointView
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v.URL != "http://192.168.1.42:3000" || v.HostIP != "192.168.1.42" {
		t.Errorf("endpoint = %+v", v)
	}
	if v.Source != "host" {
		t.Errorf("source = %q, want host", v.Source)
	}
	if !v.HostOnline {
		t.Error("host_online = false right after a push")
	}
}

// A tool with nowhere to point says so, rather than redirecting to a URL that
// cannot be parsed.
func TestGoRefusesAToolWithNoEndpoint(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	createTool(t, ts, admin, toolInput{Name: "Nowhere", SourceType: "manual", Visibility: "public"})

	resp, data := ts.do(t, noRedirect(nil), http.MethodGet, "/go/nowhere", nil, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409: %s", resp.StatusCode, data)
	}
}

// The redirect must not become a way to find out which tools exist.
func TestGoHidesToolsTheCallerCannotSee(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	createTool(t, ts, admin, toolInput{
		Name: "Secret", SourceType: "manual", Visibility: "restricted",
		Address: "10.0.0.9", Scheme: "http",
	})

	anon, _ := ts.do(t, noRedirect(nil), http.MethodGet, "/go/secret", nil, nil)
	if anon.StatusCode != http.StatusNotFound {
		t.Errorf("anonymous status = %d, want 404", anon.StatusCode)
	}
	unknown, _ := ts.do(t, noRedirect(nil), http.MethodGet, "/go/no-such-tool", nil, nil)
	if unknown.StatusCode != http.StatusNotFound {
		t.Errorf("unknown slug status = %d, want 404", unknown.StatusCode)
	}
	// The creator sees it, so the two 404s above are about visibility and not
	// about the route being broken.
	owner, _ := ts.do(t, noRedirect(admin), http.MethodGet, "/go/secret", nil, nil)
	if owner.StatusCode != http.StatusFound {
		t.Errorf("creator status = %d, want 302", owner.StatusCode)
	}
}

// A stale address is a better answer than none, so an offline host still
// redirects and says it is offline in the JSON.
func TestOfflineHostStillRedirects(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID := pushIP(t, ts, admin, "pc2", "192.168.1.42")
	createTool(t, ts, admin, toolInput{
		Name: "Grafana", SourceType: "manual", Visibility: "public",
		HostID: hostID, Scheme: "http", Port: 3000,
	})

	stale := samplePush()
	stale.IPAddress = "192.168.1.42"
	if err := ts.app.db.ApplyPush(hostID, stale, time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("ApplyPush: %v", err)
	}

	resp, _ := ts.do(t, noRedirect(nil), http.MethodGet, "/go/grafana", nil, nil)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302 even for an offline host", resp.StatusCode)
	}
	_, data := ts.do(t, nil, http.MethodGet, "/api/v1/endpoints/grafana", nil, nil)
	var v endpointView
	json.Unmarshal(data, &v)
	if v.HostOnline {
		t.Error("host_online = true for a host last seen an hour ago")
	}
}

func TestSlugCollisionsAndOverrides(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	first := createTool(t, ts, admin, toolInput{Name: "Grafana", SourceType: "manual", Visibility: "public"})
	second := createTool(t, ts, admin, toolInput{Name: "Grafana", SourceType: "manual", Visibility: "public"})
	if first.Slug != "grafana" || second.Slug != "grafana-2" {
		t.Errorf("slugs = %q, %q; want grafana, grafana-2", first.Slug, second.Slug)
	}

	// An explicitly requested slug that is taken is a 409, not a silent rename:
	// the caller is about to share that URL.
	resp, _ := ts.do(t, admin, http.MethodPost, "/api/v1/tools",
		toolInput{Name: "Third", Slug: "grafana", SourceType: "manual", Visibility: "public"}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate slug status = %d, want 409", resp.StatusCode)
	}

	resp, _ = ts.do(t, admin, http.MethodPost, "/api/v1/tools",
		toolInput{Name: "Fourth", Slug: "!!!", SourceType: "manual", Visibility: "public"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unusable slug status = %d, want 400", resp.StatusCode)
	}
}

// Renaming a tool must not move its /go/ URL: those get bookmarked and shared.
func TestRenameKeepsTheSlug(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	tool := createTool(t, ts, admin, toolInput{
		Name: "Grafana", SourceType: "manual", Visibility: "public", Address: "10.0.0.1",
	})

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/tools/"+tool.ID,
		toolInput{Name: "Metrics", SourceType: "manual", Visibility: "public", Address: "10.0.0.1"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rename = %d: %s", resp.StatusCode, data)
	}
	var updated toolResponse
	json.Unmarshal(data, &updated)
	if updated.Slug != "grafana" {
		t.Errorf("slug after rename = %q, want the original grafana", updated.Slug)
	}
}

func TestSlugCanBeChangedDeliberately(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	tool := createTool(t, ts, admin, toolInput{
		Name: "Grafana", SourceType: "manual", Visibility: "public", Address: "10.0.0.1",
	})

	resp, data := ts.do(t, admin, http.MethodPatch, "/api/v1/tools/"+tool.ID,
		toolInput{Name: "Grafana", Slug: "metrics", SourceType: "manual", Visibility: "public", Address: "10.0.0.1"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("slug change = %d: %s", resp.StatusCode, data)
	}
	if got, _ := ts.do(t, noRedirect(nil), http.MethodGet, "/go/metrics", nil, nil); got.StatusCode != http.StatusFound {
		t.Errorf("new slug status = %d, want 302", got.StatusCode)
	}
	if old, _ := ts.do(t, noRedirect(nil), http.MethodGet, "/go/grafana", nil, nil); old.StatusCode != http.StatusNotFound {
		t.Errorf("old slug status = %d, want 404", old.StatusCode)
	}
}

// The agent's reported address travels the real push path, not just the store.
func TestIngestStoresTheReportedAddress(t *testing.T) {
	ts := newTestServer(t)
	admin := adminClient(t, ts)
	hostID := pushIP(t, ts, admin, "pc2", "192.168.1.42")

	_, data := ts.do(t, admin, http.MethodGet, "/api/v1/hosts", nil, nil)
	var hosts []hostView
	json.Unmarshal(data, &hosts)
	for _, h := range hosts {
		if h.ID == hostID {
			if h.IPAddress != "192.168.1.42" {
				t.Errorf("host ip_address = %q, want 192.168.1.42", h.IPAddress)
			}
			return
		}
	}
	t.Fatalf("host %s missing from the list", hostID)
}
