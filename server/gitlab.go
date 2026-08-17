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
	"strconv"
	"strings"
)

// Settings keys for the GitLab connection, which is gitlab.com or any instance
// an operator runs themselves; only the URL tells them apart. The token is
// sealed with the master key. What to watch is not here: that is the pipeline
// groups an operator builds on their own page.
const (
	settingGitLabEnabled = "gitlab.enabled"
	settingGitLabURL     = "gitlab.url"
	settingGitLabToken   = "gitlab.token"
)

// gitlabDefaultURL is the hosted instance, which is what most connections are.
const gitlabDefaultURL = "https://gitlab.com"

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
		URL:      a.settingOr(settingGitLabURL, gitlabDefaultURL),
		TokenSet: hasToken,
	}
}

// saveGitLabSettings validates and persists the connection. An enabled
// connection must be complete, so the pipelines page never has to explain a
// half-filled form.
func (a *app) saveGitLabSettings(in gitlabInput) error {
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if in.URL == "" {
		in.URL = gitlabDefaultURL
	}
	if in.Enabled {
		if err := validateForgeURL(in.URL, "GitLab", gitlabDefaultURL); err != nil {
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

// validateForgeURL rejects anything that is not an absolute http(s) base, since
// every call appends a path to it.
func validateForgeURL(raw, label, example string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New(label + " URL must be absolute, such as " + example)
	}
	return nil
}

func (a *app) gitlabConfig() (forgeConfig, bool) {
	if !a.db.GetBoolSetting(settingGitLabEnabled, false) {
		return forgeConfig{}, false
	}
	v := a.gitlabView()
	token, _ := a.sealedSetting(settingGitLabToken)
	cfg := forgeConfig{Provider: providerGitLab, URL: v.URL, Token: token, WebURL: v.URL}
	if validateForgeURL(cfg.URL, "GitLab", gitlabDefaultURL) != nil || cfg.Token == "" {
		return forgeConfig{}, false
	}
	return cfg, true
}

// gitlabSearch asks GitLab which projects match. Ordered by last activity so an
// empty box still opens on the repos someone is most likely to be looking for.
func gitlabSearch(ctx context.Context, cfg forgeConfig, query string) ([]repoProject, error) {
	path := fmt.Sprintf("/api/v4/projects?simple=true&order_by=last_activity_at&per_page=%d&search=%s",
		searchLimit, url.QueryEscape(query))

	var found []struct {
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	}
	if err := forgeGet(ctx, cfg, path, &found); err != nil {
		return nil, err
	}
	out := make([]repoProject, 0, len(found))
	for _, p := range found {
		out = append(out, repoProject{Name: p.Name, Path: p.PathWithNamespace, URL: p.WebURL})
	}
	return out, nil
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

// gitlabLatest reads the latest pipeline of every given project path in one
// GraphQL call.
func gitlabLatest(ctx context.Context, cfg forgeConfig, paths []string) ([]repoProject, string) {
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

	resp, err := (&http.Client{Timeout: forgeTimeout}).Do(req)
	if err != nil {
		return nil, "could not reach GitLab: " + err.Error()
	}
	defer resp.Body.Close()
	if err := forgeStatusError(providerGitLab, resp.StatusCode); err != nil {
		return nil, err.Error()
	}

	var parsed gitlabGraphQLResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&parsed); err != nil {
		return nil, "could not read GitLab's reply: " + err.Error()
	}
	if len(parsed.Errors) > 0 && len(parsed.Data) == 0 {
		return nil, parsed.Errors[0].Message
	}

	out := make([]repoProject, 0, len(paths))
	for i, path := range paths {
		node := parsed.Data[fmt.Sprintf("p%d", i)]
		if node == nil {
			out = append(out, repoProject{
				Name:  path[strings.LastIndex(path, "/")+1:],
				Path:  path,
				Error: "no repo at this path, or the token cannot see it",
			})
			continue
		}
		p := repoProject{Name: node.Name, Path: node.FullPath, URL: node.WebURL}
		if len(node.Pipelines.Nodes) > 0 {
			latest := node.Pipelines.Nodes[0]
			p.Status = strings.ToLower(latest.Status)
			p.Ref = latest.Ref
			p.UpdatedAt = latest.UpdatedAt
			if latest.Path != "" {
				p.PipelineURL = cfg.WebURL + latest.Path
			}
		}
		out = append(out, p)
	}
	return out, ""
}
