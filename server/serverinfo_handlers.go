package main

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/thehelvijs/Reeve/agent/collect"
)

func (a *app) handleServerInfo(w http.ResponseWriter, _ *http.Request) {
	tools, _ := a.db.CountTools()
	hosts, _ := a.db.CountHosts()
	users, _ := a.db.CountUsers()

	var dbSize int64
	if fi, err := os.Stat(a.cfg.DBPath); err == nil {
		dbSize = fi.Size()
	}

	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":     a.cfg.Version,
		"started_at":  a.startedAt.Format(time.RFC3339),
		"uptime_secs": int64(now.Sub(a.startedAt).Seconds()),
		"go": map[string]any{
			"version":       runtime.Version(),
			"os":            runtime.GOOS,
			"arch":          runtime.GOARCH,
			"num_cpu":       runtime.NumCPU(),
			"num_goroutine": runtime.NumGoroutine(),
		},
		"db": map[string]any{
			"path":           a.cfg.DBPath,
			"size_bytes":     dbSize,
			"restore_staged": a.restoreStaged(),
		},
		"counts": map[string]any{
			"tools": tools,
			"hosts": hosts,
			"users": users,
		},
		"host": collect.SampleHostMetrics(),
	})
}
