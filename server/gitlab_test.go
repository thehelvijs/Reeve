package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gitlabStub answers the one GraphQL query the overview makes, recording the
// Authorization header so the test can prove the token never has to leave the
// server to reach GitLab.
func gitlabStub(t *testing.T, reply string) (*httptest.Server, *string) {
	t.Helper()
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/graphql" {
			t.Errorf("path = %s, want /api/graphql", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv, &auth
}

const gitlabTwoProjects = `{"data":{"group":{"webUrl":"https://git.example.com/groups/firmware","projects":{
  "pageInfo":{"hasNextPage":true},
  "nodes":[
    {"name":"pitch-encoder","fullPath":"firmware/pitch-encoder","webUrl":"https://git.example.com/firmware/pitch-encoder",
     "pipelines":{"nodes":[{"status":"SUCCESS","ref":"main","updatedAt":"2026-08-17T10:00:00Z","path":"/firmware/pitch-encoder/-/pipelines/7"}]}},
    {"name":"Powerboard5","fullPath":"firmware/Powerboard5","webUrl":"https://git.example.com/firmware/Powerboard5",
     "pipelines":{"nodes":[{"status":"FAILED","ref":"develop","updatedAt":"2026-08-17T09:00:00Z","path":"/firmware/Powerboard5/-/pipelines/9"}]}},
    {"name":"atmel-sam-s70","fullPath":"firmware/atmel-sam-s70","webUrl":"https://git.example.com/firmware/atmel-sam-s70",
     "pipelines":{"nodes":[]}}
  ]}}}}`

type pipelinesReply struct {
	Configured bool          `json:"configured"`
	Groups     []gitlabGroup `json:"groups"`
}

func getPipelines(t *testing.T, ts *testServer, c *http.Client) pipelinesReply {
	t.Helper()
	resp, data := ts.do(t, c, http.MethodGet, "/api/gitlab/pipelines", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	var out pipelinesReply
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
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

func TestGitLabPipelinesReportsLatestPerProject(t *testing.T) {
	gl, auth := gitlabStub(t, gitlabTwoProjects)
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	resp, v := putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"enabled": true, "url": gl.URL, "groups": "firmware", "token": "glpat-secret",
	}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, want 200", resp.StatusCode)
	}
	if !v.GitLab.TokenSet || v.GitLab.Groups != "firmware" {
		t.Fatalf("settings view = %+v", v.GitLab)
	}
	// The token is write-only: it is never handed back to a browser.
	if body, _ := json.Marshal(v.GitLab); strings.Contains(string(body), "glpat-secret") {
		t.Error("settings view leaks the access token")
	}

	out := getPipelines(t, ts, c)
	if !out.Configured || len(out.Groups) != 1 {
		t.Fatalf("reply = %+v", out)
	}
	g := out.Groups[0]
	if g.Error != "" {
		t.Fatalf("group error = %q", g.Error)
	}
	if !g.Truncated {
		t.Error("truncated = false, want true when GitLab reports another page")
	}
	if len(g.Projects) != 3 {
		t.Fatalf("projects = %d, want 3", len(g.Projects))
	}
	// Failed first, then by name; the project that never ran a pipeline reports
	// an empty status rather than being dropped.
	if g.Projects[0].Name != "Powerboard5" || g.Projects[0].Status != "failed" {
		t.Errorf("first project = %+v, want the failed one", g.Projects[0])
	}
	if g.Projects[0].PipelineURL != gl.URL+"/firmware/Powerboard5/-/pipelines/9" {
		t.Errorf("pipeline url = %q", g.Projects[0].PipelineURL)
	}
	if g.Projects[1].Name != "atmel-sam-s70" || g.Projects[1].Status != "" {
		t.Errorf("second project = %+v, want the one with no pipeline", g.Projects[1])
	}
	if g.Projects[2].Status != "success" || g.Projects[2].Ref != "main" {
		t.Errorf("third project = %+v", g.Projects[2])
	}
	if *auth != "Bearer glpat-secret" {
		t.Errorf("authorization = %q", *auth)
	}
}

// A group that GitLab cannot answer for reports its own error instead of
// failing the whole page.
func TestGitLabPipelinesGroupError(t *testing.T) {
	gl, _ := gitlabStub(t, `{"data":{"group":null}}`)
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")
	putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"enabled": true, "url": gl.URL, "groups": "nope, firmware", "token": "glpat-secret",
	}})

	out := getPipelines(t, ts, c)
	if len(out.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(out.Groups))
	}
	for _, g := range out.Groups {
		if g.Error == "" {
			t.Errorf("group %s has no error", g.Path)
		}
	}
}

func TestGitLabSettingsRejectIncomplete(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t)
	signup(t, ts, c, "boss@example.com", "password123")

	cases := []struct {
		name string
		in   map[string]any
	}{
		{"no url", map[string]any{"enabled": true, "url": "", "groups": "firmware", "token": "t"}},
		{"relative url", map[string]any{"enabled": true, "url": "gitlab.example.com", "groups": "firmware", "token": "t"}},
		{"no groups", map[string]any{"enabled": true, "url": "https://gitlab.example.com", "groups": " ", "token": "t"}},
		{"no token", map[string]any{"enabled": true, "url": "https://gitlab.example.com", "groups": "firmware", "token": ""}},
	}
	for _, tc := range cases {
		resp, _ := putSettings(t, ts, c, map[string]any{"gitlab": tc.in})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, resp.StatusCode)
		}
	}

	// Disabled config is saved as written, so an operator can fill it in over
	// more than one visit.
	resp, v := putSettings(t, ts, c, map[string]any{"gitlab": map[string]any{
		"enabled": false, "url": "https://gitlab.example.com", "groups": "firmware", "token": "",
	}})
	if resp.StatusCode != http.StatusOK || v.GitLab.URL != "https://gitlab.example.com" {
		t.Fatalf("status = %d, view = %+v", resp.StatusCode, v.GitLab)
	}
}

func TestParseGitLabGroups(t *testing.T) {
	got := parseGitLabGroups(" firmware,\n /tools/lidar/ \n\n cameras ")
	want := []string{"firmware", "tools/lidar", "cameras"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
