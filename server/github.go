package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Settings keys for the GitHub connection: github.com or a GitHub Enterprise
// Server, told apart by the API base URL. The token is sealed with the master
// key.
const (
	settingGitHubEnabled = "github.enabled"
	settingGitHubURL     = "github.url"
	settingGitHubToken   = "github.token"
)

// githubDefaultURL is the hosted API. Enterprise Server puts its own host in
// front of the same paths, as https://ghe.example.com/api/v3.
const githubDefaultURL = "https://api.github.com"

// githubSearchPages caps how far back the repo picker looks. GitHub's own search
// spans all of GitHub rather than what a token can see, so the picker walks the
// token's repositories instead, newest push first.
// ponytail: 3 pages is 300 repos; paginate the picker if an org outgrows that.
const githubSearchPages = 3

type githubView struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"`
	TokenSet bool   `json:"token_set"`
}

// githubInput is the write shape; an empty token keeps the stored one.
type githubInput struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Token   string `json:"token"`
}

func (a *app) githubView() githubView {
	_, hasToken := a.sealedSetting(settingGitHubToken)
	return githubView{
		Enabled:  a.db.GetBoolSetting(settingGitHubEnabled, false),
		URL:      a.settingOr(settingGitHubURL, githubDefaultURL),
		TokenSet: hasToken,
	}
}

// saveGitHubSettings validates and persists the connection.
func (a *app) saveGitHubSettings(in githubInput) error {
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if in.URL == "" {
		in.URL = githubDefaultURL
	}
	if in.Enabled {
		if err := validateForgeURL(in.URL, "GitHub", githubDefaultURL); err != nil {
			return err
		}
		if in.Token == "" {
			if _, ok := a.sealedSetting(settingGitHubToken); !ok {
				return errors.New("an access token that can read Actions is required")
			}
		}
	}
	writes := map[string]string{
		settingGitHubEnabled: strconv.FormatBool(in.Enabled),
		settingGitHubURL:     in.URL,
	}
	for k, v := range writes {
		if err := a.db.SetSetting(k, v); err != nil {
			return err
		}
	}
	if in.Token != "" {
		return a.setSealedSetting(settingGitHubToken, in.Token)
	}
	return nil
}

func (a *app) githubConfig() (forgeConfig, bool) {
	if !a.db.GetBoolSetting(settingGitHubEnabled, false) {
		return forgeConfig{}, false
	}
	v := a.githubView()
	token, _ := a.sealedSetting(settingGitHubToken)
	cfg := forgeConfig{
		Provider: providerGitHub, URL: v.URL, Token: token, WebURL: githubWebURL(v.URL),
	}
	if validateForgeURL(cfg.URL, "GitHub", githubDefaultURL) != nil || cfg.Token == "" {
		return forgeConfig{}, false
	}
	return cfg, true
}

// githubWebURL is where a repo lives for a human, given the API base. GitHub is
// the one forge whose API answers on a different host than its pages, and an
// Enterprise Server hangs its API off /api/v3 of the same host.
func githubWebURL(apiURL string) string {
	if apiURL == githubDefaultURL {
		return "https://github.com"
	}
	return strings.TrimSuffix(strings.TrimSuffix(apiURL, "/"), "/api/v3")
}

// githubSearch offers the token's repositories, filtered by the query. The forge
// is asked for repos ordered by last push, so an empty box opens on what someone
// touched most recently.
func githubSearch(ctx context.Context, cfg forgeConfig, query string) ([]repoProject, error) {
	needle := strings.ToLower(query)
	out := []repoProject{}
	for page := 1; page <= githubSearchPages && len(out) < searchLimit; page++ {
		var repos []struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			HTMLURL  string `json:"html_url"`
		}
		path := fmt.Sprintf("/user/repos?per_page=100&page=%d&sort=pushed&affiliation=%s",
			page, url.QueryEscape("owner,collaborator,organization_member"))
		if err := forgeGet(ctx, cfg, path, &repos); err != nil {
			return nil, err
		}
		for _, r := range repos {
			if len(out) == searchLimit {
				break
			}
			if needle == "" || strings.Contains(strings.ToLower(r.FullName), needle) {
				out = append(out, repoProject{Name: r.Name, Path: r.FullName, URL: r.HTMLURL})
			}
		}
		if len(repos) < 100 {
			break
		}
	}
	return out, nil
}

// githubRunStatus maps a workflow run onto the vocabulary the page speaks, which
// is GitLab's: one word per state across both forges, so the UI keeps one colour
// map and an operator learns one set of words.
func githubRunStatus(status, conclusion string) string {
	switch conclusion {
	case "success":
		return "success"
	case "failure", "timed_out", "startup_failure":
		return "failed"
	case "cancelled":
		return "canceled"
	case "skipped", "neutral", "stale":
		return "skipped"
	case "action_required":
		return "manual"
	}
	// No conclusion yet: the run is still going, so its status is the answer.
	if status == "in_progress" {
		return "running"
	}
	return "pending"
}

// githubLatest reads the latest Actions run of every given repo. GitHub has no
// batch query for this, so the calls run concurrently instead of as one request.
func githubLatest(ctx context.Context, cfg forgeConfig, paths []string) []repoProject {
	return fetchEach(ctx, paths, func(ctx context.Context, path string) repoProject {
		p := repoProject{
			Name: path[strings.LastIndex(path, "/")+1:],
			Path: path,
			URL:  cfg.WebURL + "/" + path,
		}
		var runs struct {
			WorkflowRuns []struct {
				Status     string `json:"status"`
				Conclusion string `json:"conclusion"`
				HeadBranch string `json:"head_branch"`
				UpdatedAt  string `json:"updated_at"`
				HTMLURL    string `json:"html_url"`
			} `json:"workflow_runs"`
		}
		if err := forgeGet(ctx, cfg, "/repos/"+path+"/actions/runs?per_page=1", &runs); err != nil {
			p.Error = err.Error()
			p.URL = ""
			return p
		}
		if len(runs.WorkflowRuns) == 0 {
			return p
		}
		latest := runs.WorkflowRuns[0]
		p.Status = githubRunStatus(latest.Status, latest.Conclusion)
		p.Ref = latest.HeadBranch
		p.UpdatedAt = latest.UpdatedAt
		p.PipelineURL = latest.HTMLURL
		return p
	})
}
