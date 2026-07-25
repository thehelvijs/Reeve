package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

const agentBinaryPrefix = "agent-linux-"

// handleAgentDownload dispatches /dl/{filename} to the binary or checksum
// handler. ServeMux wildcards must occupy a whole path segment, so the
// arch-suffixed filename is parsed here rather than matched by the router.
func (a *app) handleAgentDownload(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	rest, ok := strings.CutPrefix(filename, agentBinaryPrefix)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "agent build not available for that arch")
		return
	}
	if arch, ok := strings.CutSuffix(rest, ".sha256"); ok {
		r.SetPathValue("arch", arch)
		a.handleAgentChecksum(w, r)
		return
	}
	if arch, ok := strings.CutSuffix(rest, ".minisig"); ok {
		r.SetPathValue("arch", arch)
		a.handleAgentSignature(w, r)
		return
	}
	r.SetPathValue("arch", rest)
	a.handleAgentBinary(w, r)
}

var supportedArches = map[string]bool{
	"amd64": true, "arm64": true, "armv7": true,
	"armv6": true, "386": true, "riscv64": true,
}

func (a *app) openAgentBinary(arch string) (fs.File, bool) {
	if !supportedArches[arch] {
		return nil, false
	}
	f, err := a.agentFS.Open(agentBinaryPrefix + arch)
	if err != nil {
		return nil, false
	}
	return f, true
}

func (a *app) handleAgentBinary(w http.ResponseWriter, r *http.Request) {
	f, ok := a.openAgentBinary(r.PathValue("arch"))
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "agent build not available for that arch")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, f)
}

func (a *app) handleAgentChecksum(w http.ResponseWriter, r *http.Request) {
	arch := r.PathValue("arch")
	f, ok := a.openAgentBinary(arch)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "agent build not available for that arch")
		return
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not hash binary")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, hex.EncodeToString(h.Sum(nil))+"  "+agentBinaryPrefix+arch+"\n")
}

// handleAgentSignature serves the release signature published beside the
// embedded binary. A build with unsigned agents has none, and the agent treats
// its absence as a refusal to update rather than a reason to skip verification.
func (a *app) handleAgentSignature(w http.ResponseWriter, r *http.Request) {
	arch := r.PathValue("arch")
	if !supportedArches[arch] {
		writeError(w, http.StatusNotFound, "not_found", "agent build not available for that arch")
		return
	}
	data, err := fs.ReadFile(a.agentFS, agentBinaryPrefix+arch+".minisig")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "this build publishes no signature for that arch")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (a *app) serveScript(w http.ResponseWriter, name string) {
	data, err := fs.ReadFile(a.scriptFS, "scripts/"+name)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "script not available")
		return
	}
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Write(data)
}

func (a *app) handleInstallScript(w http.ResponseWriter, _ *http.Request) {
	a.serveScript(w, "install.sh")
}

func (a *app) handleUninstallScript(w http.ResponseWriter, _ *http.Request) {
	a.serveScript(w, "uninstall.sh")
}
