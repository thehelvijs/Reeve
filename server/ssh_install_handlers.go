package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/sshinstall"
)

// unameToArch maps a host's `uname -m` to the arch suffix the agent builds use.
var unameToArch = map[string]string{
	"x86_64":  "amd64",
	"amd64":   "amd64",
	"aarch64": "arm64",
	"arm64":   "arm64",
	"armv7l":  "armv7",
	"armv6l":  "armv6",
	"i386":    "386",
	"i686":    "386",
	"riscv64": "riscv64",
}

// sshTargetInput is the connection detail an admin supplies. None of it is
// stored: it lives for the length of the request only.
type sshTargetInput struct {
	Address      string `json:"address"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	PrivateKey   string `json:"private_key"`
	Passphrase   string `json:"passphrase"`
	SudoPassword string `json:"sudo_password"`
	Fingerprint  string `json:"fingerprint"`
}

func (in sshTargetInput) target() sshinstall.Target {
	return sshinstall.Target{
		Address:      strings.TrimSpace(in.Address),
		Port:         in.Port,
		User:         strings.TrimSpace(in.Username),
		Password:     in.Password,
		PrivateKey:   in.PrivateKey,
		Passphrase:   in.Passphrase,
		SudoPassword: in.SudoPassword,
		Fingerprint:  in.Fingerprint,
	}
}

// handleSSHProbe returns the host key fingerprint for an admin to confirm before
// any credential is sent. Trust-on-first-use, made explicit.
func (a *app) handleSSHProbe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(in.Address) == "" {
		writeError(w, http.StatusBadRequest, "invalid_address", "host address is required")
		return
	}
	fingerprint, keyType, err := sshinstall.Probe(r.Context(), strings.TrimSpace(in.Address), in.Port)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ssh_unreachable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"fingerprint": fingerprint, "key_type": keyType})
}

// handleSSHInstall pushes the agent onto a host over SSH and enrolls it. The
// enrollment token is minted here and never leaves the server: it goes straight
// into the installer's environment on the target.
func (a *app) handleSSHInstall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h, err := a.db.GetHost(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	var in sshTargetInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	target := in.target()
	if err := target.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	serverURL := a.baseURL(r)
	if a.cfg.PublicURL == "" && !reachableFromOtherHosts(serverURL) {
		writeError(w, http.StatusBadRequest, "unreachable_server_url",
			"reach this UI by the server's LAN address instead of "+serverURL+
				", or set REEVE_PUBLIC_URL, so the agent has an address it can push to")
		return
	}

	// A fresh token every push: the old one's plaintext is gone, and a
	// reinstall must not depend on anyone having kept it.
	token, hash := auth.NewAgentToken()
	if err := a.db.SetHostEnrollTokenHash(h.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not issue an enrollment token")
		return
	}

	scripts, err := a.installScriptPair()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "install scripts are not available in this build")
		return
	}
	env := map[string]string{
		"REEVE_SERVER_URL":    serverURL,
		"REEVE_AGENT_TOKEN":   token,
		"REEVE_PUSH_INTERVAL": "15s",
	}
	out, err := sshinstall.Install(r.Context(), target, a.agentBinaryForUname, scripts, env)
	if err != nil {
		log.Printf("ssh install on %s: %v", target.Address, err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"code": "install_failed", "message": err.Error(), "output": out,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"host_id": h.ID, "output": out})
}

// handleSSHUninstall removes the agent from a host over SSH, leaving the host in
// the catalog so its history stays readable.
func (a *app) handleSSHUninstall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.db.GetHost(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	var in sshTargetInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	target := in.target()
	if err := target.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	scripts, err := a.installScriptPair()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "install scripts are not available in this build")
		return
	}
	out, err := sshinstall.Uninstall(r.Context(), target, scripts)
	if err != nil {
		log.Printf("ssh uninstall on %s: %v", target.Address, err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"code": "uninstall_failed", "message": err.Error(), "output": out,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"output": out})
}

// installScriptPair reads the embedded installer and uninstaller.
func (a *app) installScriptPair() (sshinstall.Scripts, error) {
	install, err := fs.ReadFile(a.scriptFS, "scripts/install.sh")
	if err != nil {
		return sshinstall.Scripts{}, err
	}
	uninstall, err := fs.ReadFile(a.scriptFS, "scripts/uninstall.sh")
	if err != nil {
		return sshinstall.Scripts{}, err
	}
	return sshinstall.Scripts{Install: install, Uninstall: uninstall}, nil
}

// agentBinaryForUname returns the embedded agent build matching a host's
// `uname -m`, with its checksum so the remote installer can confirm the copy.
func (a *app) agentBinaryForUname(uname string) ([]byte, string, error) {
	arch, ok := unameToArch[strings.TrimSpace(uname)]
	if !ok {
		return nil, "", errUnsupportedArch(uname)
	}
	f, ok := a.openAgentBinary(arch)
	if !ok {
		return nil, "", errNoAgentBuild(arch)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}

type errUnsupportedArch string

func (e errUnsupportedArch) Error() string {
	return "unsupported host architecture " + string(e) + "; supported: x86_64 aarch64 armv7l armv6l i686 riscv64"
}

type errNoAgentBuild string

func (e errNoAgentBuild) Error() string {
	return "this build embeds no agent for " + string(e)
}
