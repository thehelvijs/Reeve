package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Settings keys for the GitLab pipeline overview. The token is sealed with the
// master key; the rest is readable config.
const (
	settingGitLabEnabled = "gitlab.enabled"
	settingGitLabURL     = "gitlab.url"
	settingGitLabGroups  = "gitlab.groups"
	settingGitLabToken   = "gitlab.token"
)

// gitlabProjectLimit is how many projects one group reports. A group past it is
// flagged truncated rather than silently cut.
const gitlabProjectLimit = 100

// gitlabTimeout bounds one GraphQL call, so an unreachable GitLab fails the
// page instead of holding a request open.
const gitlabTimeout = 15 * time.Second

type gitlabView struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"`
	Groups   string `json:"groups"`
	TokenSet bool   `json:"token_set"`
}

// gitlabInput is the write shape; an empty token keeps the stored one.
type gitlabInput struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Groups  string `json:"groups"`
	Token   string `json:"token"`
}

func (a *app) gitlabView() gitlabView {
	_, hasToken := a.sealedSetting(settingGitLabToken)
	return gitlabView{
		Enabled:  a.db.GetBoolSetting(settingGitLabEnabled, false),
		URL:      a.settingOr(settingGitLabURL, ""),
		Groups:   a.settingOr(settingGitLabGroups, ""),
		TokenSet: hasToken,
	}
}

// saveGitLabSettings validates and persists the connection. An enabled
// connection must be complete, so the pipelines page never has to explain a
// half-filled form.
func (a *app) saveGitLabSettings(in gitlabInput) error {
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	in.Groups = strings.TrimSpace(in.Groups)
	if in.Enabled {
		if err := validateGitLabURL(in.URL); err != nil {
			return err
		}
		if len(parseGitLabGroups(in.Groups)) == 0 {
			return errors.New("list at least one group path, such as firmware or firmware/lidar")
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
		settingGitLabGroups:  in.Groups,
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

// parseGitLabGroups splits the configured paths, which an operator writes one
// per line or comma separated.
func parseGitLabGroups(raw string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	}) {
		if p := strings.Trim(f, "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

type gitlabConfig struct {
	URL    string
	Token  string
	Groups []string
}

// gitlabConfig assembles the connection, reporting false when it is off or
// incomplete, so a broken config degrades to "not configured".
func (a *app) gitlabConfig() (gitlabConfig, bool) {
	if !a.db.GetBoolSetting(settingGitLabEnabled, false) {
		return gitlabConfig{}, false
	}
	v := a.gitlabView()
	token, _ := a.sealedSetting(settingGitLabToken)
	cfg := gitlabConfig{URL: v.URL, Token: token, Groups: parseGitLabGroups(v.Groups)}
	if validateGitLabURL(cfg.URL) != nil || cfg.Token == "" || len(cfg.Groups) == 0 {
		return gitlabConfig{}, false
	}
	return cfg, true
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
}

type gitlabGroup struct {
	Path     string          `json:"path"`
	URL      string          `json:"url,omitempty"`
	Projects []gitlabProject `json:"projects"`
	// Truncated says the group holds more projects than one page reports.
	Truncated bool `json:"truncated"`
	// Error is this group's own failure. One unreachable or misspelled group
	// does not take the rest of the page down with it.
	Error string `json:"error,omitempty"`
}

// One call per group returns every project under it, including subgroups, with
// its latest pipeline attached. The REST API would need a request per project.
const gitlabPipelineQuery = `query($path: ID!, $limit: Int!) {
  group(fullPath: $path) {
    webUrl
    projects(includeSubgroups: true, first: $limit) {
      pageInfo { hasNextPage }
      nodes {
        name
        fullPath
        webUrl
        pipelines(first: 1) {
          nodes { status ref updatedAt path }
        }
      }
    }
  }
}`

type gitlabGraphQLResponse struct {
	Data struct {
		Group *struct {
			WebURL   string `json:"webUrl"`
			Projects struct {
				PageInfo struct {
					HasNextPage bool `json:"hasNextPage"`
				} `json:"pageInfo"`
				Nodes []struct {
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
				} `json:"nodes"`
			} `json:"projects"`
		} `json:"group"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// fetchGitLabGroup reads one group's projects and their latest pipelines. Every
// failure comes back on the group rather than as an error, so the page renders
// whatever else it could reach.
func fetchGitLabGroup(ctx context.Context, cfg gitlabConfig, path string) gitlabGroup {
	g := gitlabGroup{Path: path, Projects: []gitlabProject{}}
	body, err := json.Marshal(map[string]any{
		"query":     gitlabPipelineQuery,
		"variables": map[string]any{"path": path, "limit": gitlabProjectLimit},
	})
	if err != nil {
		g.Error = err.Error()
		return g
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL+"/api/graphql", bytes.NewReader(body))
	if err != nil {
		g.Error = err.Error()
		return g
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := (&http.Client{Timeout: gitlabTimeout}).Do(req)
	if err != nil {
		g.Error = "could not reach GitLab: " + err.Error()
		return g
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		g.Error = "GitLab rejected the access token"
		return g
	}
	if resp.StatusCode != http.StatusOK {
		g.Error = "GitLab returned HTTP " + strconv.Itoa(resp.StatusCode)
		return g
	}

	var parsed gitlabGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		g.Error = "could not read GitLab's reply: " + err.Error()
		return g
	}
	if len(parsed.Errors) > 0 {
		g.Error = parsed.Errors[0].Message
		return g
	}
	if parsed.Data.Group == nil {
		g.Error = "no group at this path, or the token cannot see it"
		return g
	}

	g.URL = parsed.Data.Group.WebURL
	g.Truncated = parsed.Data.Group.Projects.PageInfo.HasNextPage
	for _, p := range parsed.Data.Group.Projects.Nodes {
		proj := gitlabProject{Name: p.Name, Path: p.FullPath, URL: p.WebURL}
		if len(p.Pipelines.Nodes) > 0 {
			latest := p.Pipelines.Nodes[0]
			proj.Status = strings.ToLower(latest.Status)
			proj.Ref = latest.Ref
			proj.UpdatedAt = latest.UpdatedAt
			if latest.Path != "" {
				proj.PipelineURL = cfg.URL + latest.Path
			}
		}
		g.Projects = append(g.Projects, proj)
	}
	sortGitLabProjects(g.Projects)
	return g
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

// handleGitLabPipelines reports every configured group's projects and their
// latest pipeline. An unconfigured instance answers 200 with configured:false,
// so the page can point at settings instead of showing an error.
func (a *app) handleGitLabPipelines(w http.ResponseWriter, r *http.Request) {
	cfg, ok := a.gitlabConfig()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "groups": []gitlabGroup{}})
		return
	}
	groups := make([]gitlabGroup, 0, len(cfg.Groups))
	for _, path := range cfg.Groups {
		groups = append(groups, fetchGitLabGroup(r.Context(), cfg, path))
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": true, "groups": groups})
}
