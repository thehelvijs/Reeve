package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

// Settings keys for the Coolify connection: the instance an operator runs and a
// token from its Keys & Tokens page. The token is sealed with the master key.
const (
	settingCoolifyURL   = "coolify.url"
	settingCoolifyToken = "coolify.token"
)

// coolifyDefaultURL is where Coolify answers on the machine it was installed on,
// which is the machine Reeve is usually watching. The server reaches it over the
// host network, so loopback is the address that works without asking anyone.
const coolifyDefaultURL = "http://127.0.0.1:8000"

// coolifyPaths are the resource lists worth naming. Each is optional: an
// instance that does not answer one still names what it does answer.
var coolifyPaths = []string{"/api/v1/applications", "/api/v1/services", "/api/v1/databases"}

// coolifyTimeout bounds one call. Nothing waits on this: a slow instance costs a
// push its names, not the push.
const coolifyTimeout = 5 * time.Second

// coolifyMaxBody caps one response. Generous on purpose: see coolifyGet.
const coolifyMaxBody = 64 << 20

// coolifyTTL is how long a fetched name list stands. Deploy names change when
// someone renames a resource, which is rare enough that a minute of staleness
// costs nothing and saves a call per push.
const coolifyTTL = time.Minute

type coolifyView struct {
	URL      string `json:"url"`
	TokenSet bool   `json:"token_set"`
}

// coolifyInput is the write shape; an empty token keeps the stored one.
type coolifyInput struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func (a *app) coolifyView() coolifyView {
	_, hasToken := a.sealedSetting(settingCoolifyToken)
	return coolifyView{URL: a.settingOr(settingCoolifyURL, coolifyDefaultURL), TokenSet: hasToken}
}

// saveCoolifySettings validates and persists the connection. The token is the
// connection: an empty URL means the usual one, not off, so pasting a token is
// the whole setup on the machine Coolify was installed on.
func (a *app) saveCoolifySettings(in coolifyInput) error {
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if in.URL == "" {
		in.URL = coolifyDefaultURL
	}
	if err := validateForgeURL(in.URL, "Coolify", coolifyDefaultURL); err != nil {
		return err
	}
	if err := a.db.SetSetting(settingCoolifyURL, in.URL); err != nil {
		return err
	}
	if in.Token != "" {
		return a.setSealedSetting(settingCoolifyToken, in.Token)
	}
	return nil
}

func (a *app) coolifyConfig() (url, token string, ok bool) {
	v := a.coolifyView()
	token, _ = a.sealedSetting(settingCoolifyToken)
	if token == "" {
		return "", "", false
	}
	return v.URL, token, true
}

// coolifyResource is one deployable Coolify knows about.
type coolifyResource struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// coolifyNames caches uuid -> name for one instance, refreshed on demand.
//
// A field on app rather than a package global so a test gets its own, and so two
// instances of the server never share one.
type coolifyNames struct {
	mu       sync.Mutex
	names    map[string]string
	fetched  time.Time
	lastErr  error
	inflight bool
}

// coolifyResourceNames returns the current uuid -> name map, refetching when the
// cached one has aged out. An error leaves the previous map in place: a Coolify
// that went away should not rename every container back to its uuid.
func (a *app) coolifyResourceNames(ctx context.Context) map[string]string {
	base, token, ok := a.coolifyConfig()
	if !ok {
		return nil
	}
	c := &a.coolify
	c.mu.Lock()
	fresh := time.Since(c.fetched) < coolifyTTL
	if fresh || c.inflight {
		defer c.mu.Unlock()
		return c.names
	}
	c.inflight = true
	c.mu.Unlock()

	names, lists, err := fetchCoolifyNames(ctx, base, token)
	// The push path has nobody to report to, so a list that refused says so in the
	// log rather than only under a button an admin has to press.
	for _, l := range lists {
		if l.Error != "" {
			log.Printf("coolify: %s: %s", l.Path, l.Error)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.inflight = false
	c.fetched = time.Now()
	c.lastErr = err
	if err == nil {
		c.names = names
	}
	return c.names
}

// fetchCoolifyNames asks one instance for every resource it can name. A path
// that errors is skipped; only a total failure is an error, since an instance
// with no databases answers that list with a 404 on some versions.
func fetchCoolifyNames(ctx context.Context, base, token string) (map[string]string, []coolifyList, error) {
	names := map[string]string{}
	lists := make([]coolifyList, 0, len(coolifyPaths))
	var lastErr error
	for _, path := range coolifyPaths {
		var found []coolifyResource
		if err := coolifyGet(ctx, base, token, path, &found); err != nil {
			lastErr = err
			lists = append(lists, coolifyList{Path: path, Error: err.Error()})
			continue
		}
		named := 0
		for _, r := range found {
			if r.UUID != "" && r.Name != "" {
				names[r.UUID] = r.Name
				named++
			}
		}
		lists = append(lists, coolifyList{Path: path, Found: named})
	}
	if len(names) == 0 && lastErr != nil {
		return nil, lists, lastErr
	}
	return names, lists, nil
}

// coolifyList is what one resource list answered, so a report can say which one
// came back empty rather than only how many names there are altogether.
type coolifyList struct {
	Path  string `json:"path"`
	Found int    `json:"found"`
	Error string `json:"error,omitempty"`
}

func coolifyGet(ctx context.Context, base, token, path string, dst any) error {
	ctx, cancel := context.WithTimeout(ctx, coolifyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: coolifyTimeout}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.New("Coolify rejected the token")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Coolify returned %d for %s", resp.StatusCode, path)
	}
	// Streamed against a generous cap rather than read whole: one application
	// carries its destination, its server and that server's entire proxy
	// configuration, so a handful of them runs to megabytes, and a cap that
	// truncated the JSON would fail the decode and quietly name nothing.
	return json.NewDecoder(io.LimitReader(resp.Body, coolifyMaxBody)).Decode(dst)
}

// nameCoolifyContainers gives every container Coolify deployed the name Coolify
// calls it.
//
// The join is the uuid: Coolify builds a container name out of the resource uuid
// plus its own parts — the uuid alone for a database, "<uuid>-proxy" for what
// fronts it, "<service>-<uuid>-<deploy>" for a compose service. Matching the uuid
// as one of those parts rather than anywhere in the string keeps a name that
// merely contains it from being claimed.
//
// A resource with several containers on one machine would otherwise put the same
// name on all of them, so those keep the part that tells them apart. A resource
// with one container reads as its plain name, which is the common case and the
// point of the exercise.
//
// What Coolify's API says beats what a label said. The labels are read by the
// agent with no connection to ask, and one of them carries the same generated
// string as the container name — so honouring a label here left the uuid on
// screen with the real name one call away.
func nameCoolifyContainers(containers []contracts.ContainerState, names map[string]string) {
	if len(names) == 0 {
		return
	}
	matched := make([]string, len(containers))
	perResource := map[string]int{}
	for i := range containers {
		for _, part := range strings.Split(containers[i].Name, "-") {
			if _, ok := names[part]; ok {
				matched[i] = part
				perResource[part]++
				break
			}
		}
	}
	for i, uuid := range matched {
		if uuid == "" {
			continue
		}
		name := names[uuid]
		if perResource[uuid] > 1 {
			if role := containerRole(containers[i].Name, uuid); role != "" {
				name += " (" + role + ")"
			}
		}
		containers[i].DisplayName = name
		containers[i].ManagedBy = "coolify"
	}
}

// containerRole is what is left of a container name once the resource uuid and
// the deploy number are taken out: the compose service, or the sidecar's job.
func containerRole(container, uuid string) string {
	var parts []string
	for _, part := range strings.Split(container, "-") {
		if part == uuid || part == "" || isAllDigits(part) {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "-")
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// renameProcessContainers re-points each process at whatever its container ended
// up called.
//
// The agent names a process's container from what docker told it, which is the
// uuid-shaped name when only Coolify knows better. The containers have just been
// renamed, so the processes have to follow or the two lists disagree about the
// same container.
func renameProcessContainers(procs []contracts.ProcessSample, containers []contracts.ContainerState) {
	better := make(map[string]string, len(containers))
	for _, c := range containers {
		if c.DisplayName != "" && c.DisplayName != c.Name {
			better[c.Name] = c.DisplayName
		}
	}
	if len(better) == 0 {
		return
	}
	for i := range procs {
		if name, ok := better[procs[i].Container]; ok {
			procs[i].Container = name
		}
	}
}

// handleTestCoolify reports what the stored connection can see, so a wrong URL
// or a token without the read ability surfaces at setup rather than as names
// that never improve.
func (a *app) handleTestCoolify(w http.ResponseWriter, r *http.Request) {
	base, token, ok := a.coolifyConfig()
	if !ok {
		writeError(w, http.StatusBadRequest, "no_coolify", "save a Coolify URL and token first")
		return
	}
	names, lists, err := fetchCoolifyNames(r.Context(), base, token)
	if err != nil {
		writeError(w, http.StatusBadGateway, "coolify_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url":       base,
		"resources": len(names),
		"lists":     lists,
	})
}
