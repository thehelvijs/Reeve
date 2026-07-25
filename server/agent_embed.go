package main

import (
	"embed"
	"io/fs"
)

// agentFiles holds the cross-compiled agent binaries. Populated by
// `python3 scripts/release.py --embed`; the committed PLACEHOLDER keeps
// go:embed and `go build` working when no binaries are present.
//
//go:embed all:agentdist
var agentFiles embed.FS

// installScripts holds the host install/uninstall scripts, staged into
// server/scripts by `make server-assets` (source of truth: deploy/*.sh).
//
//go:embed all:scripts
var installScripts embed.FS

// agentDistFS returns the agentdist subtree.
func agentDistFS() fs.FS {
	sub, err := fs.Sub(agentFiles, "agentdist")
	if err != nil {
		panic(err)
	}
	return sub
}
