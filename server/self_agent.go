package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// selfAgentTokenFile is read by the agent shipped beside the server, so the
// machine Reeve runs on is monitored by an agent like any other with nothing for
// an operator to enroll.
const selfAgentTokenFile = "self-agent-token"

// selfAgentTokenPath puts the token beside the database, the one directory an
// operator already has to persist, so the agent's mount follows the server's own
// volume rather than a second path to keep in step.
func (a *app) selfAgentTokenPath() string {
	return filepath.Join(filepath.Dir(a.cfg.DBPath), selfAgentTokenFile)
}

// syncSelfAgentToken makes the file and the server host's token hash agree.
//
// The file is the only record of the plaintext — the row keeps a hash — so an
// existing file wins and its hash is simply restated. That is what carries the
// pair across a restored backup or a rebuilt row, where the hash would otherwise
// stand for a token nobody holds. An admin who issues a token for this host by
// hand takes the file with them, and their token then survives every restart.
func (a *app) syncSelfAgentToken() error {
	path := a.selfAgentTokenPath()
	if b, err := os.ReadFile(path); err == nil {
		if token := strings.TrimSpace(string(b)); token != "" {
			return a.db.SetHostEnrollTokenHash(store.ServerHostID, auth.HashToken(token))
		}
	}
	token, hash := auth.NewAgentToken()
	if err := writeSecretFile(path, token+"\n"); err != nil {
		return err
	}
	return a.db.SetHostEnrollTokenHash(store.ServerHostID, hash)
}

// writeSecretFile writes atomically and readable by root alone: this one grants
// an agent the right to report as a host, and it lives in a volume other
// containers mount.
func writeSecretFile(path, content string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.WriteString(content)
	if err := tmp.Close(); err != nil || writeErr != nil {
		os.Remove(tmpPath)
		if writeErr != nil {
			return writeErr
		}
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
