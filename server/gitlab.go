package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Settings keys for the GitLab connection. The token is sealed with the master
// key; the rest is readable config. What to watch is not here: that is the
// pipeline groups an operator builds on their own page.
const (
	settingGitLabEnabled = "gitlab.enabled"
	settingGitLabURL     = "gitlab.url"
	settingGitLabToken   = "gitlab.token"
)

// gitlabSearchLimit is how many projects one search offers. A picker is for
// recognising the repo you meant, not for browsing every repo there is.
const gitlabSearchLimit = 20

// gitlabTimeout bounds one call to GitLab, so an unreachable server fails the
// page instead of holding a request open.
const gitlabTimeout = 15 * time.Second

type gitlabView struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"`
	TokenSet bool   `json:"token_set"`
}

// gitlabInput is the write shape; an empty token keeps the stored one.
type gitlabInput struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Token   string `json:"token"`
}

func (a *app) gitlabView() gitlabView {
	_, hasToken := a.sealedSetting(settingGitLabToken)
	return gitlabView{
		Enabled:  a.db.GetBoolSetting(settingGitLabEnabled, false),
		URL:      a.settingOr(settingGitLabURL, ""),
		TokenSet: hasToken,
	}
}

// saveGitLabSettings validates and persists the connection. An enabled
// connection must be complete, so the pipelines page never has to explain a
// half-filled form.
func (a *app) saveGitLabSettings(in gitlabInput) error {
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if in.Enabled {
		if err := validateGitLabURL(in.URL); err != nil {
			return err
		}
		if in.Token == "" {
			if _, ok := a.sealedSetting(settingGitLabToken); !ok {
				return errors.New("an access token with the read_api scope is required")
			}
		}
	}
	writes := map[string]string{
		settingGitLabEnabled: strconv.FormatBool(in.Enabled),
		settingGitLabURL:     in.URL,
	}
	for k, v := range writes {
		if err := a.db.SetSetting(k, v); err != nil {
			return err
		}
	}
	if in.Token != "" {
		return a.setSealedSetting(settingGitLabToken, in.Token)
	}
	return nil
}

func validateGitLabURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("GitLab URL must be absolute, such as https://gitlab.example.com")
	}
	return nil
}

type gitlabConfig struct {
	URL   string
	Token string
}

// gitlabConfig assembles the connection, reporting false when it is off or
// incomplete, so a broken config degrades to "not configured".
func (a *app) gitlabConfig() (gitlabConfig, bool) {
	if !a.db.GetBoolSetting(settingGitLabEnabled, false) {
		return gitlabConfig{}, false
	}
	v := a.gitlabView()
	token, _ := a.sealedSetting(settingGitLabToken)
	cfg := gitlabConfig{URL: v.URL, Token: token}
	if validateGitLabURL(cfg.URL) != nil || cfg.Token == "" {
		return gitlabConfig{}, false
	}
	return cfg, true
}

// gitlabGet calls a GitLab REST endpoint with the stored token. The token stays
// on this side: a browser never talks to GitLab directly, which is also what
// keeps the page working under the connect-src 'self' policy.
func gitlabGet(ctx context.Context, cfg gitlabConfig, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := (&http.Client{Timeout: gitlabTimeout}).Do(req)
	if err != nil {
		return errors.New("could not reach GitLab: " + err.Error())
	}
	defer resp.Body.Close()
	if err := gitlabStatusError(resp.StatusCode); err != nil {
		return err
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(dst)
}

func gitlabStatusError(status int) error {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return errors.New("GitLab rejected the access token")
	}
	if status != http.StatusOK {
		return errors.New("GitLab returned HTTP " + strconv.Itoa(status))
	}
	return nil
}

type gitlabProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
	URL  string `json:"url"`
	// Status is the latest pipeline's, lowercased; empty when the project has
	// never run one.
	Status      string `json:"status"`
	Ref         string `json:"ref,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	PipelineURL string `json:"pipeline_url,omitempty"`
	// Error is set when this project alone could not be read, so one renamed or
	// deleted repo does not hide the rest of its group.
	Error string `json:"error,omitempty"`
}

type pipelineGroupView struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Projects []gitlabProject `json:"projects"`
	// Error is a failure that took the whole group down, such as GitLab being
	// unreachable.
	Error string `json:"error,omitempty"`
}

// handleSearchGitLabProjects offers projects to add to a group. GitLab's own
// search decides what matches, so a repo is found the way its owner would look
// for it rather than by exact path.
func (a *app) handleSearchGitLabProjects(w http.ResponseWriter, r *http.Request) {
	cfg, ok := a.gitlabConfig()
	if !ok {
		writeError(w, http.StatusPreconditionFailed, "gitlab_not_configured",
			"connect a GitLab server in Settings first")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	// Ordered by last activity so an empty box still opens on the repos someone
	// is most likely to be looking for.
	path := fmt.Sprintf("/api/v4/projects?simple=true&order_by=last_activity_at&per_page=%d&search=%s",
		gitlabSearchLimit, url.QueryEscape(q))

	var found []struct {
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	}
	if err := gitlabGet(r.Context(), cfg, path, &found); err != nil {
		writeError(w, http.StatusBadGateway, "gitlab_failed", err.Error())
		return
	}
	out := make([]gitlabProject, 0, len(found))
	for _, p := range found {
		out = append(out, gitlabProject{Name: p.Name, Path: p.PathWithNamespace, URL: p.WebURL})
	}
	writeJSON(w, http.StatusOK, out)
}

// One call per group returns every member project with its latest pipeline.
// Aliased fields rather than a bulk filter: a group is an arbitrary set of
// paths, and aliases are the one way to ask for exactly those in a single
// request on every GitLab version.
const gitlabProjectFragment = `fragment latest on Project {
  name
  fullPath
  webUrl
  pipelines(first: 1) { nodes { status ref updatedAt path } }
}`

type gitlabProjectNode struct {
	Name      string `json:"name"`
	FullPath  string `json:"fullPath"`
	WebURL    string `json:"webUrl"`
	Pipelines struct {
		Nodes []struct {
			Status    string `json:"status"`
			Ref       string `json:"ref"`
			UpdatedAt string `json:"updatedAt"`
			Path      string `json:"path"`
		} `json:"nodes"`
	} `json:"pipelines"`
}

type gitlabGraphQLResponse struct {
	Data   map[string]*gitlabProjectNode `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// buildPipelineQuery asks for the given paths in one request, each under its own
// alias. Paths are GraphQL string literals, so they are marshalled rather than
// quoted by hand.
func buildPipelineQuery(paths []string) string {
	var b strings.Builder
	b.WriteString("query {\n")
	for i, p := range paths {
		literal, _ := json.Marshal(p)
		fmt.Fprintf(&b, "  p%d: project(fullPath: %s) { ...latest }\n", i, literal)
	}
	b.WriteString("}\n")
	b.WriteString(gitlabProjectFragment)
	return b.String()
}

// fetchPipelines reads the latest pipeline of every given project path. Every
// failure comes back on the group rather than as an error, so the page renders
// whatever else it could reach.
func fetchPipelines(ctx context.Context, cfg gitlabConfig, paths []string) ([]gitlabProject, string) {
	if len(paths) == 0 {
		return []gitlabProject{}, ""
	}
	body, err := json.Marshal(map[string]any{"query": buildPipelineQuery(paths)})
	if err != nil {
		return nil, err.Error()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL+"/api/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := (&http.Client{Timeout: gitlabTimeout}).Do(req)
	if err != nil {
		return nil, "could not reach GitLab: " + err.Error()
	}
	defer resp.Body.Close()
	if err := gitlabStatusError(resp.StatusCode); err != nil {
		return nil, err.Error()
	}

	var parsed gitlabGraphQLResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&parsed); err != nil {
		return nil, "could not read GitLab's reply: " + err.Error()
	}
	if len(parsed.Errors) > 0 && len(parsed.Data) == 0 {
		return nil, parsed.Errors[0].Message
	}

	out := make([]gitlabProject, 0, len(paths))
	for i, path := range paths {
		node := parsed.Data[fmt.Sprintf("p%d", i)]
		if node == nil {
			out = append(out, gitlabProject{
				Name:  path[strings.LastIndex(path, "/")+1:],
				Path:  path,
				Error: "no project at this path, or the token cannot see it",
			})
			continue
		}
		p := gitlabProject{Name: node.Name, Path: node.FullPath, URL: node.WebURL}
		if len(node.Pipelines.Nodes) > 0 {
			latest := node.Pipelines.Nodes[0]
			p.Status = strings.ToLower(latest.Status)
			p.Ref = latest.Ref
			p.UpdatedAt = latest.UpdatedAt
			if latest.Path != "" {
				p.PipelineURL = cfg.URL + latest.Path
			}
		}
		out = append(out, p)
	}
	sortGitLabProjects(out)
	return out, ""
}

// sortGitLabProjects puts what is broken first: the page exists to be scanned
// for red, not read alphabetically.
func sortGitLabProjects(projects []gitlabProject) {
	sort.SliceStable(projects, func(i, j int) bool {
		if (projects[i].Status == "failed") != (projects[j].Status == "failed") {
			return projects[i].Status == "failed"
		}
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
}

// handleGitLabPipelines reports every pipeline group with its projects' latest
// pipeline. An unconfigured instance answers 200 with configured:false, so the
// page can point at settings instead of showing an error.
func (a *app) handleGitLabPipelines(w http.ResponseWriter, r *http.Request) {
	groups, err := a.db.ListPipelineGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read pipeline groups")
		return
	}
	cfg, ok := a.gitlabConfig()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "groups": []pipelineGroupView{}})
		return
	}
	out := make([]pipelineGroupView, 0, len(groups))
	for _, g := range groups {
		projects, groupErr := fetchPipelines(r.Context(), cfg, g.Projects)
		if projects == nil {
			projects = []gitlabProject{}
		}
		out = append(out, pipelineGroupView{ID: g.ID, Name: g.Name, Projects: projects, Error: groupErr})
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": true, "groups": out})
}
