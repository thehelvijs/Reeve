package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// githubStub answers the two calls the GitHub side makes: the repo list behind
// the picker, and one Actions run per repo.
type githubStub struct {
	srv   *httptest.Server
	auth  string
	asked []string
}

func newGitHubStub(t *testing.T) *githubStub {
	t.Helper()
	s := &githubStub{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/user/repos":
			_, _ = w.Write([]byte(`[
			  {"name":"reeve","full_name":"acme/reeve","html_url":"https://github.com/acme/reeve"},
			  {"name":"pitch-encoder","full_name":"acme/pitch-encoder","html_url":"https://github.com/acme/pitch-encoder"}
			]`))
		case strings.HasSuffix(r.URL.Path, "/actions/runs"):
			repo := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/repos/"), "/actions/runs")
			s.asked = append(s.asked, repo)
			switch repo {
			case "acme/reeve":
				_, _ = w.Write([]byte(`{"workflow_runs":[{"status":"completed","conclusion":"failure",
				  "head_branch":"main","updated_at":"2026-08-17T09:00:00Z",
				  "html_url":"https://github.com/acme/reeve/actions/runs/42"}]}`))
			case "acme/quiet":
				_, _ = w.Write([]byte(`{"workflow_runs":[]}`))
			case "acme/gone":
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"Not Found"}`))
			default:
				_, _ = w.Write([]byte(`{"workflow_runs":[{"status":"in_progress","conclusion":null,
				  "head_branch":"topic","updated_at":"2026-08-17T10:00:00Z",
				  "html_url":"https://github.com/` + repo + `/actions/runs/7"}]}`))
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

// connectGitHub points the instance at the stub and returns an admin client.
func connectGitHub(t *testing.T, ts *testServer, stub *githubStub) *http.Client {
	t.Helper()
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	resp, v := putSettings(t, ts, c, map[string]any{"github": map[string]any{
		"url": stub.srv.URL, "token": "ghp-secret",
	}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, want 200", resp.StatusCode)
	}
	if !v.GitHub.TokenSet {
		t.Fatal("token_set = false after saving a token")
	}
	if body, _ := json.Marshal(v.GitHub); strings.Contains(string(body), "ghp-secret") {
		t.Error("settings view leaks the access token")
	}
	return c
}

func createProviderGroup(t *testing.T, ts *testServer, c *http.Client, name, provider string) string {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/pipeline-groups",
		map[string]any{"name": name, "provider": provider}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201: %s", resp.StatusCode, data)
	}
	var g pipelineGroupRecord
	json.Unmarshal(data, &g)
	return g.ID
}

func TestGitHubPipelinesReportsActionsRuns(t *testing.T) {
	stub := newGitHubStub(t)
	ts := newTestServer(t)
	c := connectGitHub(t, ts, stub)

	id := createProviderGroup(t, ts, c, "Tooling", "github")
	for _, p := range []string{"acme/reeve", "acme/pitch-encoder", "acme/quiet", "acme/gone"} {
		if resp := addProject(t, ts, c, id, p); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("add %s status = %d", p, resp.StatusCode)
		}
	}

	out := getPipelines(t, ts, c)
	if !out.Configured || len(out.Groups) != 1 {
		t.Fatalf("reply = %+v", out)
	}
	g := out.Groups[0]
	if g.Provider != "github" || g.Error != "" {
		t.Fatalf("group = %+v", g)
	}
	byPath := map[string]repoProject{}
	for _, p := range g.Projects {
		byPath[p.Path] = p
	}
	if len(byPath) != 4 {
		t.Fatalf("projects = %+v", g.Projects)
	}
	// A completed run with conclusion failure reads as failed, and sorts first.
	if g.Projects[0].Path != "acme/reeve" || g.Projects[0].Status != "failed" {
		t.Errorf("first project = %+v, want the failed one", g.Projects[0])
	}
	if byPath["acme/reeve"].PipelineURL != "https://github.com/acme/reeve/actions/runs/42" {
		t.Errorf("run url = %q", byPath["acme/reeve"].PipelineURL)
	}
	// A run with no conclusion yet is in flight, not green.
	if byPath["acme/pitch-encoder"].Status != "running" {
		t.Errorf("in-progress status = %q, want running", byPath["acme/pitch-encoder"].Status)
	}
	if byPath["acme/quiet"].Status != "" {
		t.Errorf("never-run status = %q, want empty", byPath["acme/quiet"].Status)
	}
	if byPath["acme/gone"].Error == "" {
		t.Error("a repo GitHub answers 404 for carries no error")
	}
	if stub.auth != "Bearer ghp-secret" {
		t.Errorf("authorization = %q", stub.auth)
	}
	if len(stub.asked) != 4 {
		t.Errorf("asked for %d repos, want 4", len(stub.asked))
	}
}

func TestGitHubRepoSearchFilters(t *testing.T) {
	stub := newGitHubStub(t)
	ts := newTestServer(t)
	c := connectGitHub(t, ts, stub)

	resp, data := ts.do(t, c, http.MethodGet, "/api/admin/repo-search?provider=github&q=encoder", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var found []repoProject
	json.Unmarshal(data, &found)
	if len(found) != 1 || found[0].Path != "acme/pitch-encoder" {
		t.Fatalf("found = %+v", found)
	}

	// An empty box lists what the token can see, most recently pushed first.
	resp, data = ts.do(t, c, http.MethodGet, "/api/admin/repo-search?provider=github&q=", nil, nil)
	json.Unmarshal(data, &found)
	if resp.StatusCode != http.StatusOK || len(found) != 2 {
		t.Fatalf("empty search = %d %+v", resp.StatusCode, found)
	}
}

// A group on a provider nobody has connected says so, instead of rendering as
// an empty group or failing the page.
func TestPipelineGroupOnUnconnectedProvider(t *testing.T) {
	stub := newGitHubStub(t)
	ts := newTestServer(t)
	c := connectGitHub(t, ts, stub)
	id := createProviderGroup(t, ts, c, "Firmware", "gitlab")
	addProject(t, ts, c, id, "firmware/Powerboard5")

	out := getPipelines(t, ts, c)
	if len(out.Groups) != 1 || !strings.Contains(out.Groups[0].Error, "GitLab is not connected") {
		t.Fatalf("groups = %+v", out.Groups)
	}
}

// Both forges at once: the provider belongs to the group, not the instance.
func TestPipelinesMixesProviders(t *testing.T) {
	gh := newGitHubStub(t)
	gl := newGitLabStub(t, gitlabAliasedReply, gitlabSearchReply)
	ts := newTestServer(t)
	c := connectGitHub(t, ts, gh)
	putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"url": gl.srv.URL, "token": "glpat-secret",
	}})

	ghGroup := createProviderGroup(t, ts, c, "Tooling", "github")
	addProject(t, ts, c, ghGroup, "acme/reeve")
	glGroup := createProviderGroup(t, ts, c, "Firmware", "gitlab")
	addProject(t, ts, c, glGroup, "firmware/Powerboard5")

	out := getPipelines(t, ts, c)
	if len(out.Groups) != 2 {
		t.Fatalf("groups = %+v", out.Groups)
	}
	seen := map[string]string{}
	for _, g := range out.Groups {
		if g.Error != "" {
			t.Errorf("group %s error = %q", g.Name, g.Error)
		}
		seen[g.Name] = g.Provider
	}
	if seen["Tooling"] != "github" || seen["Firmware"] != "gitlab" {
		t.Errorf("providers = %+v", seen)
	}
}

func TestGitHubRunStatusMapping(t *testing.T) {
	cases := []struct{ status, conclusion, want string }{
		{"completed", "success", "success"},
		{"completed", "failure", "failed"},
		{"completed", "timed_out", "failed"},
		{"completed", "startup_failure", "failed"},
		{"completed", "cancelled", "canceled"},
		{"completed", "skipped", "skipped"},
		{"completed", "neutral", "skipped"},
		{"completed", "action_required", "manual"},
		{"in_progress", "", "running"},
		{"queued", "", "pending"},
		{"waiting", "", "pending"},
	}
	for _, tc := range cases {
		if got := githubRunStatus(tc.status, tc.conclusion); got != tc.want {
			t.Errorf("githubRunStatus(%q, %q) = %q, want %q", tc.status, tc.conclusion, got, tc.want)
		}
	}
}

// The API host is not the web host on GitHub, and an Enterprise Server hangs its
// API off a path of the same host.
func TestGitHubWebURL(t *testing.T) {
	cases := map[string]string{
		"https://api.github.com":            "https://github.com",
		"https://ghe.example.com/api/v3":    "https://ghe.example.com",
		"https://ghe.example.com":           "https://ghe.example.com",
	}
	for api, want := range cases {
		if got := githubWebURL(api); got != want {
			t.Errorf("githubWebURL(%q) = %q, want %q", api, got, want)
		}
	}
}

func TestGitHubSettingsRejectRelativeURL(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := putSettings(t, ts, c, map[string]any{"github": map[string]any{
		"url": "api.github.com", "token": "t",
	}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("relative url status = %d, want 400", resp.StatusCode)
	}

	// An empty URL defaults to the hosted API rather than failing, so the common
	// case is one token and nothing else.
	resp, v := putSettings(t, ts, c, map[string]any{"github": map[string]any{
		"url": "", "token": "ghp-x",
	}})
	if resp.StatusCode != http.StatusOK || v.GitHub.URL != "https://api.github.com" {
		t.Fatalf("status = %d, view = %+v", resp.StatusCode, v.GitHub)
	}
}

func TestCreateGroupRejectsUnknownProvider(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/pipeline-groups",
		map[string]any{"name": "Nope", "provider": "bitbucket"}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
