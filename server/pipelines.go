package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The forges a pipeline group can point at. Either or both may be connected: the
// provider belongs to the group, so firmware on GitLab and tooling on GitHub
// show up on one page.
const (
	providerGitLab = "gitlab"
	providerGitHub = "github"
)

// searchLimit is how many repos one search offers. A picker is for recognising
// the repo you meant, not for browsing every repo there is.
const searchLimit = 20

// forgeTimeout bounds one call to a forge, so an unreachable server fails the
// page instead of holding a request open.
const forgeTimeout = 15 * time.Second

// forgeConcurrency caps how many repo calls run at once for providers with no
// batch query. Enough to keep a group of twenty quick, low enough that a page
// refresh is not mistaken for an attack.
const forgeConcurrency = 8

func validProvider(p string) bool {
	return p == providerGitLab || p == providerGitHub
}

// providerLabel is the forge's own name, for a message an operator reads.
func providerLabel(p string) string {
	if p == providerGitHub {
		return "GitHub"
	}
	return "GitLab"
}

// forgeConfig is one connected forge: where it is and the token to read it with.
type forgeConfig struct {
	Provider string
	// URL is the API base, without a trailing slash.
	URL   string
	Token string
	// WebURL is where a repo lives for a human. It is the same host for GitLab
	// and a different one for GitHub, whose API has a host of its own.
	WebURL string
}

// forge returns the connection for one provider, reporting false when it is off
// or incomplete, so a broken config degrades to "not connected".
func (a *app) forge(provider string) (forgeConfig, bool) {
	if provider == providerGitHub {
		return a.githubConfig()
	}
	return a.gitlabConfig()
}

// repoProject is one repo and the state of its latest run, whatever the forge
// calls that: a GitLab pipeline or a GitHub Actions workflow run.
type repoProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
	URL  string `json:"url"`
	// Status is lowercased and uses GitLab's vocabulary for both forges, so one
	// status has one meaning and the UI needs one colour map: success, failed,
	// running, pending, canceled, skipped, manual. Empty means never run.
	Status      string `json:"status"`
	Ref         string `json:"ref,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	PipelineURL string `json:"pipeline_url,omitempty"`
	// Error is set when this repo alone could not be read, so one renamed or
	// deleted repo does not hide the rest of its group.
	Error string `json:"error,omitempty"`
}

type pipelineGroupView struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Provider string        `json:"provider"`
	Projects []repoProject `json:"projects"`
	// Error is a failure that took the whole group down, such as its forge being
	// unreachable or not connected.
	Error string `json:"error,omitempty"`
}

// forgeGet calls a forge's REST API with the stored token. The token stays on
// this side: a browser never talks to the forge directly, which is also what
// keeps the page working under the connect-src 'self' policy.
func forgeGet(ctx context.Context, cfg forgeConfig, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	if cfg.Provider == providerGitHub {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	resp, err := (&http.Client{Timeout: forgeTimeout}).Do(req)
	if err != nil {
		return errors.New("could not reach " + providerLabel(cfg.Provider) + ": " + err.Error())
	}
	defer resp.Body.Close()
	if err := forgeStatusError(cfg.Provider, resp.StatusCode); err != nil {
		return err
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(dst)
}

func forgeStatusError(provider string, status int) error {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return errors.New(providerLabel(provider) + " rejected the access token")
	}
	if status == http.StatusNotFound {
		return errors.New("no repo at this path, or the token cannot see it")
	}
	if status != http.StatusOK {
		return errors.New(providerLabel(provider) + " returned HTTP " + strconv.Itoa(status))
	}
	return nil
}

// searchRepos offers repos to add to a group, using the forge's own idea of what
// matches so a repo is found the way its owner would look for it.
func searchRepos(ctx context.Context, cfg forgeConfig, query string) ([]repoProject, error) {
	if cfg.Provider == providerGitHub {
		return githubSearch(ctx, cfg, query)
	}
	return gitlabSearch(ctx, cfg, query)
}

// fetchLatest reads the latest run of every given path. It returns a group-wide
// error only for a failure that says nothing about the individual repos; a repo
// that alone could not be read carries its own.
func fetchLatest(ctx context.Context, cfg forgeConfig, paths []string) ([]repoProject, string) {
	if len(paths) == 0 {
		return []repoProject{}, ""
	}
	if cfg.Provider == providerGitHub {
		return githubLatest(ctx, cfg, paths), ""
	}
	return gitlabLatest(ctx, cfg, paths)
}

// fetchEach runs one call per path with a bounded pool, for a forge with no
// batch query. Results keep the order of paths so the page does not reshuffle
// on every refresh for reasons nobody can see.
func fetchEach(ctx context.Context, paths []string, one func(context.Context, string) repoProject) []repoProject {
	out := make([]repoProject, len(paths))
	slots := make(chan struct{}, forgeConcurrency)
	var wg sync.WaitGroup
	for i, path := range paths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			out[i] = one(ctx, path)
		}(i, path)
	}
	wg.Wait()
	return out
}

// sortRepoProjects puts what is broken first: the page exists to be scanned for
// red, not read alphabetically.
func sortRepoProjects(projects []repoProject) {
	sort.SliceStable(projects, func(i, j int) bool {
		if (projects[i].Status == "failed") != (projects[j].Status == "failed") {
			return projects[i].Status == "failed"
		}
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
}

// pipelinesConfigured reports whether any forge is connected, which is what
// decides between an empty page and a "connect one" prompt.
func (a *app) pipelinesConfigured() bool {
	for _, p := range []string{providerGitLab, providerGitHub} {
		if _, ok := a.forge(p); ok {
			return true
		}
	}
	return false
}

// handleSearchRepos offers repos on one provider for the group editor.
func (a *app) handleSearchRepos(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if !validProvider(provider) {
		writeError(w, http.StatusBadRequest, "invalid_provider", "provider must be gitlab or github")
		return
	}
	cfg, ok := a.forge(provider)
	if !ok {
		writeError(w, http.StatusPreconditionFailed, "forge_not_configured",
			"connect "+providerLabel(provider)+" in Settings first")
		return
	}
	found, err := searchRepos(r.Context(), cfg, strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		writeError(w, http.StatusBadGateway, "forge_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, found)
}

// handlePipelines reports every group with its repos' latest run. An instance
// with nothing connected answers 200 with configured:false, so the page can
// point at settings instead of showing an error.
func (a *app) handlePipelines(w http.ResponseWriter, r *http.Request) {
	groups, err := a.db.ListPipelineGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read pipeline groups")
		return
	}
	configured := a.pipelinesConfigured()
	out := make([]pipelineGroupView, 0, len(groups))
	for _, g := range groups {
		view := pipelineGroupView{ID: g.ID, Name: g.Name, Provider: g.Provider, Projects: []repoProject{}}
		cfg, ok := a.forge(g.Provider)
		if !ok {
			view.Error = providerLabel(g.Provider) + " is not connected; add it in Settings"
			out = append(out, view)
			continue
		}
		projects, groupErr := fetchLatest(r.Context(), cfg, g.Projects)
		if projects != nil {
			sortRepoProjects(projects)
			view.Projects = projects
		}
		view.Error = groupErr
		out = append(out, view)
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": configured, "groups": out})
}
