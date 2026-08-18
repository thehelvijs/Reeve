package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/thehelvijs/Reeve/contracts"
)

// coolifyStub answers the three resource lists, counting the calls so the cache
// can be checked.
func coolifyStub(t *testing.T, calls *atomic.Int64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		calls.Add(1)
		var body []map[string]string
		switch r.URL.Path {
		case "/api/v1/applications":
			body = []map[string]string{{"uuid": "oy26vjo0k3r6yxdiu2fywqzt", "name": "billing-api"}}
		case "/api/v1/services":
			body = []map[string]string{{"uuid": "svc9911", "name": "mailpit"}}
		default:
			// Some versions answer a list they have nothing for with a 404, which
			// must not lose the names the other lists gave.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The whole point of the connection: a container named after a uuid reads as the
// application Coolify calls it.
func TestCoolifyNamesContainersByUUID(t *testing.T) {
	var calls atomic.Int64
	stub := coolifyStub(t, &calls)
	ts := newTestServer(t)
	if err := ts.app.saveCoolifySettings(coolifyInput{URL: stub.URL, Token: "tok-1"}); err != nil {
		t.Fatal(err)
	}

	names := ts.app.coolifyResourceNames(context.Background())
	if names["oy26vjo0k3r6yxdiu2fywqzt"] != "billing-api" || names["svc9911"] != "mailpit" {
		t.Fatalf("names = %v", names)
	}

	containers := []contracts.ContainerState{
		{ID: "c1", Name: "web-oy26vjo0k3r6yxdiu2fywqzt-112139724840"},
		{ID: "c2", Name: "coolify-db"},
		{ID: "c3", Name: "web-svc9911-99", DisplayName: "named-by-label", ManagedBy: "coolify"},
	}
	nameCoolifyContainers(containers, names)
	if containers[0].DisplayName != "billing-api" || containers[0].ManagedBy != "coolify" {
		t.Errorf("uuid container = %+v", containers[0])
	}
	if containers[1].DisplayName != "" || containers[1].ManagedBy != "" {
		t.Errorf("Coolify's own container was renamed: %+v", containers[1])
	}
	if containers[2].DisplayName != "named-by-label" {
		t.Errorf("a label already named this one: %+v", containers[2])
	}

	// Cached: a second read inside the TTL asks Coolify nothing.
	before := calls.Load()
	ts.app.coolifyResourceNames(context.Background())
	if calls.Load() != before {
		t.Errorf("calls went from %d to %d inside the cache window", before, calls.Load())
	}
}

// A connection that breaks must not rename every container back to its uuid: the
// last good answer stands.
func TestCoolifyKeepsTheLastNamesWhenItGoesAway(t *testing.T) {
	var calls atomic.Int64
	stub := coolifyStub(t, &calls)
	ts := newTestServer(t)
	if err := ts.app.saveCoolifySettings(coolifyInput{URL: stub.URL, Token: "tok-1"}); err != nil {
		t.Fatal(err)
	}
	if got := ts.app.coolifyResourceNames(context.Background()); len(got) != 2 {
		t.Fatalf("names = %v", got)
	}

	stub.Close()
	ts.app.coolify.fetched = ts.app.coolify.fetched.Add(-2 * coolifyTTL)
	if got := ts.app.coolifyResourceNames(context.Background()); len(got) != 2 {
		t.Errorf("names after Coolify went away = %v, want the previous two", got)
	}
}

// A bad token is the mistake worth reporting at setup time, not silently.
func TestTestCoolifyReportsARejectedToken(t *testing.T) {
	var calls atomic.Int64
	stub := coolifyStub(t, &calls)
	ts := newTestServer(t)
	admin := adminClient(t, ts)

	resp, _ := ts.do(t, admin, http.MethodPost, "/api/admin/settings/test-coolify", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("test with nothing saved = %d, want 400", resp.StatusCode)
	}

	if err := ts.app.saveCoolifySettings(coolifyInput{URL: stub.URL, Token: "wrong"}); err != nil {
		t.Fatal(err)
	}
	resp, _ = ts.do(t, admin, http.MethodPost, "/api/admin/settings/test-coolify", nil, nil)
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("test with a rejected token = %d, want 502", resp.StatusCode)
	}

	if err := ts.app.saveCoolifySettings(coolifyInput{URL: stub.URL, Token: "tok-1"}); err != nil {
		t.Fatal(err)
	}
	resp, data := ts.do(t, admin, http.MethodPost, "/api/admin/settings/test-coolify", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test with a good token = %d: %s", resp.StatusCode, data)
	}
	var out struct{ Resources int }
	json.Unmarshal(data, &out)
	if out.Resources != 2 {
		t.Errorf("resources = %d, want 2", out.Resources)
	}
}

// A process says which container it runs in, and the agent could only name that
// container what docker called it. Renaming one list and not the other leaves two
// names for one container on the same page.
func TestRenameProcessContainersFollowsTheContainer(t *testing.T) {
	containers := []contracts.ContainerState{
		{ID: "c1", Name: "web-oy26vjo0k3r6yxdiu2fywqzt-112139724840", DisplayName: "Web app"},
		{ID: "c2", Name: "coolify-db"},
	}
	procs := []contracts.ProcessSample{
		{PID: 1, Command: "php-fpm", Container: "web-oy26vjo0k3r6yxdiu2fywqzt-112139724840"},
		{PID: 2, Command: "postgres", Container: "coolify-db"},
		{PID: 3, Command: "sshd"},
	}
	renameProcessContainers(procs, containers)
	if procs[0].Container != "Web app" {
		t.Errorf("renamed container = %q, want Web app", procs[0].Container)
	}
	if procs[1].Container != "coolify-db" {
		t.Errorf("unrenamed container = %q, want coolify-db", procs[1].Container)
	}
	if procs[2].Container != "" {
		t.Errorf("host process = %q, want empty", procs[2].Container)
	}
}

// A database's container is the bare uuid and what fronts it is "<uuid>-proxy", so
// one resource can own several containers on one machine. They must not all read
// as the same name, and a resource with one container must not pick up a suffix
// it does not need.
func TestCoolifyNamesSiblingContainersApart(t *testing.T) {
	names := map[string]string{
		"ijcxohcv4zrsijh0ripypsvm": "posgresql",
		"oy26vjo0k3r6yxdiu2fywqzt": "Web app",
	}
	containers := []contracts.ContainerState{
		{ID: "c1", Name: "ijcxohcv4zrsijh0ripypsvm"},
		{ID: "c2", Name: "ijcxohcv4zrsijh0ripypsvm-proxy"},
		{ID: "c3", Name: "web-oy26vjo0k3r6yxdiu2fywqzt-112139724840"},
		{ID: "c4", Name: "unrelated-ijcxohcv4zrsijh0ripypsvmx"},
	}
	nameCoolifyContainers(containers, names)

	got := map[string]string{}
	for _, c := range containers {
		got[c.ID] = c.DisplayName
	}
	want := map[string]string{
		"c1": "posgresql",
		"c2": "posgresql (proxy)",
		"c3": "Web app",
		// The uuid is a substring of this one's name, not a part of it.
		"c4": "",
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("container %s named %q, want %q", id, got[id], w)
		}
	}
}
