package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gitlabStub answers the two calls the pipelines feature makes: the aliased
// GraphQL query behind the overview, and the REST project search behind the
// picker. It records what it was asked so a test can prove the token never has
// to leave the server, and that the query really names the group's projects.
type gitlabStub struct {
	srv     *httptest.Server
	auth    string
	query   string
	search  string
	graphql string
}

func newGitLabStub(t *testing.T, graphqlReply, searchReply string) *gitlabStub {
	t.Helper()
	s := &gitlabStub{graphql: graphqlReply}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/graphql":
			body, _ := io.ReadAll(r.Body)
			s.query = string(body)
			_, _ = w.Write([]byte(s.graphql))
		case "/api/v4/projects":
			s.search = r.URL.Query().Get("search")
			_, _ = w.Write([]byte(searchReply))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

// Aliases follow the group's stored order, which is by path: p0 is
// firmware/Powerboard5, p1 is firmware/gone and p2 is firmware/pitch-encoder.
// p1 comes back null, as GitLab answers for a path the token cannot see.
const gitlabAliasedReply = `{"data":{
  "p0":{"name":"Powerboard5","fullPath":"firmware/Powerboard5","webUrl":"https://git.example.com/firmware/Powerboard5",
        "pipelines":{"nodes":[{"status":"FAILED","ref":"develop","updatedAt":"2026-08-17T09:00:00Z","path":"/firmware/Powerboard5/-/pipelines/9"}]}},
  "p1":null,
  "p2":{"name":"pitch-encoder","fullPath":"firmware/pitch-encoder","webUrl":"https://git.example.com/firmware/pitch-encoder",
        "pipelines":{"nodes":[{"status":"SUCCESS","ref":"main","updatedAt":"2026-08-17T10:00:00Z","path":"/firmware/pitch-encoder/-/pipelines/7"}]}}}}`

const gitlabSearchReply = `[
  {"name":"pitch-encoder","path_with_namespace":"firmware/pitch-encoder","web_url":"https://git.example.com/firmware/pitch-encoder"},
  {"name":"Powerboard5","path_with_namespace":"firmware/Powerboard5","web_url":"https://git.example.com/firmware/Powerboard5"}
]`

type pipelinesReply struct {
	Configured bool                `json:"configured"`
	Groups     []pipelineGroupView `json:"groups"`
}

func getPipelines(t *testing.T, ts *testServer, c *http.Client) pipelinesReply {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodGet, "/api/pipelines", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var out pipelinesReply
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// connectGitLab points the instance at the stub and returns an admin client.
func connectGitLab(t *testing.T, ts *testServer, stub *gitlabStub) *http.Client {
	t.Helper()
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	resp, v := putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"url": stub.srv.URL, "token": "glpat-secret",
	}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, want 200", resp.StatusCode)
	}
	if !v.GitLab.TokenSet {
		t.Fatal("token_set = false after saving a token")
	}
	if body, _ := json.Marshal(v.GitLab); strings.Contains(string(body), "glpat-secret") {
		t.Error("settings view leaks the access token")
	}
	return c
}

// createGroup makes a group and returns its id.
func createGroup(t *testing.T, ts *testServer, c *http.Client, name string) string {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodPost, "/api/admin/pipeline-groups", map[string]any{"name": name, "provider": "gitlab"}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201: %s", resp.StatusCode, data)
	}
	var g pipelineGroupRecord
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return g.ID
}

func addProject(t *testing.T, ts *testServer, c *http.Client, id, path string) *http.Response {
	t.Helper()
	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/pipeline-groups/"+id+"/projects",
		map[string]any{"path": path}, nil)
	return resp
}

func TestGitLabPipelinesUnconfigured(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	out := getPipelines(t, ts, c)
	if out.Configured {
		t.Error("configured = true with no GitLab settings saved")
	}
}

func TestPipelineGroupCRUD(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	id := createGroup(t, ts, c, "Firmware")

	// A name is taken once, so two groups cannot answer to the same thing.
	resp, _ := ts.do(t, c, http.MethodPost, "/api/admin/pipeline-groups", map[string]any{"name": "Firmware", "provider": "gitlab"}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate name status = %d, want 409", resp.StatusCode)
	}

	if resp = addProject(t, ts, c, id, "firmware/Powerboard5"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add status = %d, want 204", resp.StatusCode)
	}
	// Adding twice is the same as adding once.
	if resp = addProject(t, ts, c, id, "firmware/Powerboard5"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("re-add status = %d, want 204", resp.StatusCode)
	}
	if resp = addProject(t, ts, c, id, "Powerboard5"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bare path status = %d, want 400", resp.StatusCode)
	}
	if resp = addProject(t, ts, c, "nosuchgroup", "firmware/x"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown group status = %d, want 404", resp.StatusCode)
	}

	resp, data := ts.do(t, c, http.MethodGet, "/api/pipeline-groups", nil, nil)
	var groups []pipelineGroupRecord
	json.Unmarshal(data, &groups)
	if resp.StatusCode != http.StatusOK || len(groups) != 1 || len(groups[0].Projects) != 1 {
		t.Fatalf("list = %d %+v", resp.StatusCode, groups)
	}

	resp, _ = ts.do(t, c, http.MethodPatch, "/api/admin/pipeline-groups/"+id, map[string]any{"name": "Cameras"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("rename status = %d, want 204", resp.StatusCode)
	}

	resp, _ = ts.do(t, c, http.MethodDelete, "/api/admin/pipeline-groups/"+id+"/projects",
		map[string]any{"path": "firmware/Powerboard5"}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove status = %d, want 204", resp.StatusCode)
	}

	resp, _ = ts.do(t, c, http.MethodDelete, "/api/admin/pipeline-groups/"+id, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", resp.StatusCode)
	}
	resp, data = ts.do(t, c, http.MethodGet, "/api/pipeline-groups", nil, nil)
	json.Unmarshal(data, &groups)
	if len(groups) != 0 {
		t.Errorf("groups after delete = %+v", groups)
	}
}

// Deleting a group takes its membership with it, so a later group of the same
// name does not inherit projects nobody added to it.
func TestPipelineGroupDeleteCascades(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	id := createGroup(t, ts, c, "Firmware")
	addProject(t, ts, c, id, "firmware/Powerboard5")
	ts.do(t, c, http.MethodDelete, "/api/admin/pipeline-groups/"+id, nil, nil)

	n, err := ts.app.db.CountPipelineGroupProjects(id)
	if err != nil || n != 0 {
		t.Errorf("membership after delete = %d (err %v), want 0", n, err)
	}
}

func TestPipelineGroupsAdminOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t)
	signup(t, ts, admin, "boss@example.com", "password123")
	id := createGroup(t, ts, admin, "Firmware")

	basic := ts.client(t)
	signup(t, ts, basic, "dev@example.com", "password123")

	// Reading is open to the team; writing is not.
	resp, _ := ts.do(t, basic, http.MethodGet, "/api/pipeline-groups", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("basic list status = %d, want 200", resp.StatusCode)
	}
	if resp = addProject(t, ts, basic, id, "firmware/x"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic add status = %d, want 403", resp.StatusCode)
	}
	resp, _ = ts.do(t, basic, http.MethodPost, "/api/admin/pipeline-groups", map[string]any{"name": "Nope", "provider": "gitlab"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic create status = %d, want 403", resp.StatusCode)
	}
	resp, _ = ts.do(t, basic, http.MethodGet, "/api/admin/repo-search?provider=gitlab&q=x", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("basic search status = %d, want 403", resp.StatusCode)
	}
}

func TestGitLabPipelinesReportsEachGroupsProjects(t *testing.T) {
	stub := newGitLabStub(t, gitlabAliasedReply, gitlabSearchReply)
	ts := newTestServer(t)
	c := connectGitLab(t, ts, stub)

	id := createGroup(t, ts, c, "Firmware")
	for _, p := range []string{"firmware/Powerboard5", "firmware/pitch-encoder", "firmware/gone"} {
		if resp := addProject(t, ts, c, id, p); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("add %s status = %d", p, resp.StatusCode)
		}
	}

	out := getPipelines(t, ts, c)
	if !out.Configured || len(out.Groups) != 1 {
		t.Fatalf("reply = %+v", out)
	}
	g := out.Groups[0]
	if g.Name != "Firmware" || g.Error != "" {
		t.Fatalf("group = %+v", g)
	}
	if len(g.Projects) != 3 {
		t.Fatalf("projects = %d, want 3", len(g.Projects))
	}
	// Failed first, then by name.
	if g.Projects[0].Name != "Powerboard5" || g.Projects[0].Status != "failed" {
		t.Errorf("first project = %+v, want the failed one", g.Projects[0])
	}
	if g.Projects[0].PipelineURL != stub.srv.URL+"/firmware/Powerboard5/-/pipelines/9" {
		t.Errorf("pipeline url = %q", g.Projects[0].PipelineURL)
	}
	// The project GitLab would not answer for reports its own error and keeps
	// its place, rather than silently vanishing from the group.
	var gone *repoProject
	for i := range g.Projects {
		if g.Projects[i].Path == "firmware/gone" {
			gone = &g.Projects[i]
		}
	}
	if gone == nil || gone.Error == "" {
		t.Errorf("missing project = %+v, want one carrying an error", gone)
	}

	// The query names every member of the group, and carries the token.
	for _, want := range []string{"firmware/Powerboard5", "firmware/pitch-encoder", "firmware/gone"} {
		if !strings.Contains(stub.query, want) {
			t.Errorf("query does not name %s", want)
		}
	}
	if stub.auth != "Bearer glpat-secret" {
		t.Errorf("authorization = %q", stub.auth)
	}
}

// A group nobody has put a project in yet renders as empty rather than as an
// error, and costs no call to GitLab.
func TestGitLabPipelinesEmptyGroup(t *testing.T) {
	stub := newGitLabStub(t, gitlabAliasedReply, gitlabSearchReply)
	ts := newTestServer(t)
	c := connectGitLab(t, ts, stub)
	createGroup(t, ts, c, "Empty")

	out := getPipelines(t, ts, c)
	if len(out.Groups) != 1 || len(out.Groups[0].Projects) != 0 || out.Groups[0].Error != "" {
		t.Fatalf("groups = %+v", out.Groups)
	}
	if stub.query != "" {
		t.Errorf("an empty group called GitLab: %q", stub.query)
	}
}

func TestGitLabProjectSearch(t *testing.T) {
	stub := newGitLabStub(t, gitlabAliasedReply, gitlabSearchReply)
	ts := newTestServer(t)
	c := connectGitLab(t, ts, stub)

	resp, data := ts.do(t, c, http.MethodGet, "/api/admin/repo-search?provider=gitlab&q=encoder", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var found []repoProject
	if err := json.Unmarshal(data, &found); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(found) != 2 || found[0].Path != "firmware/pitch-encoder" {
		t.Fatalf("found = %+v", found)
	}
	if stub.search != "encoder" {
		t.Errorf("search term forwarded as %q", stub.search)
	}
}

func TestGitLabProjectSearchNeedsAConnection(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := ts.do(t, c, http.MethodGet, "/api/admin/repo-search?provider=gitlab&q=x", nil, nil)
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want 412", resp.StatusCode)
	}
}

// GitLab being unreachable is the group's error, not a failed page.
func TestGitLabPipelinesGroupError(t *testing.T) {
	stub := newGitLabStub(t, `{"errors":[{"message":"something broke"}]}`, gitlabSearchReply)
	ts := newTestServer(t)
	c := connectGitLab(t, ts, stub)
	id := createGroup(t, ts, c, "Firmware")
	addProject(t, ts, c, id, "firmware/Powerboard5")

	out := getPipelines(t, ts, c)
	if len(out.Groups) != 1 || out.Groups[0].Error != "something broke" {
		t.Fatalf("groups = %+v", out.Groups)
	}
}

func TestGitLabSettingsRejectRelativeURL(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, _ := putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"url": "gitlab.example.com", "token": "t",
	}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("relative url status = %d, want 400", resp.StatusCode)
	}

	// A URL without a token is saved as written, so an operator can fill the
	// connection in over more than one visit.
	resp, v := putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"url": "https://gitlab.example.com", "token": "",
	}})
	if resp.StatusCode != http.StatusOK || v.GitLab.URL != "https://gitlab.example.com" {
		t.Fatalf("status = %d, view = %+v", resp.StatusCode, v.GitLab)
	}

	// An empty URL is the hosted instance, not an error: gitlab.com is a GitLab
	// like any other and only the URL tells them apart.
	resp, v = putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"url": "", "token": "glpat-x",
	}})
	if resp.StatusCode != http.StatusOK || v.GitLab.URL != "https://gitlab.com" {
		t.Fatalf("status = %d, view = %+v", resp.StatusCode, v.GitLab)
	}
}

func TestBuildPipelineQueryNamesEveryPath(t *testing.T) {
	q := buildPipelineQuery([]string{"a/b", `we"ird/path`})
	if !strings.Contains(q, `p0: project(fullPath: "a/b")`) {
		t.Errorf("query = %s", q)
	}
	// A path is a string literal, so a quote in it cannot end the literal early.
	if !strings.Contains(q, `p1: project(fullPath: "we\"ird/path")`) {
		t.Errorf("query = %s", q)
	}
}
