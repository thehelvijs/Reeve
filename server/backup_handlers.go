package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// maxRestoreBytes caps an uploaded database. Backups are the whole instance, so
// the ceiling is generous but not unbounded.
const maxRestoreBytes = 2 << 30

// handleBackupDownload streams a consistent copy of the database. The copy is
// still encrypted where the live database is: credentials stay ciphertext and
// the master key is not part of it, so a backup alone reveals no secret.
func (a *app) handleBackupDownload(w http.ResponseWriter, r *http.Request) {
	dir, err := os.MkdirTemp("", "reeve-backup-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not stage a backup")
		return
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "backup.db")
	if err := a.db.BackupTo(path); err != nil {
		log.Printf("backup: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not copy the database")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not read the backup")
		return
	}
	defer f.Close()

	name := "reeve-" + a.cfg.Version + "-" + time.Now().UTC().Format("20060102-150405") + ".db"
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	if fi, err := f.Stat(); err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	}
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("backup: stream to client: %v", err)
	}
}

// handleRestoreUpload stages an uploaded database. It is validated before it is
// staged and only swapped in on the next start, so a restore cannot corrupt the
// database this process has open.
func (a *app) handleRestoreUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRestoreBytes+4096)
	file, _, err := formFile(r, "backup")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "a database file is required")
		return
	}
	defer file.Close()

	tmp := store.StagedRestorePath(a.cfg.DBPath) + ".tmp"
	defer os.Remove(tmp)
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not write the upload")
		return
	}
	written, copyErr := io.Copy(dst, io.LimitReader(file, maxRestoreBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not write the upload")
		return
	}
	if written > maxRestoreBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "database exceeds the 2 GB limit")
		return
	}
	if err := store.ValidateBackup(tmp); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_backup", err.Error())
		return
	}
	if err := os.Rename(tmp, store.StagedRestorePath(a.cfg.DBPath)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not stage the restore")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"staged":      true,
		"size_bytes":  written,
		"applies_on":  "next server start",
		"instruction": "restart Reeve to swap this database in",
	})
}

// handleRestoreCancel discards a staged restore.
func (a *app) handleRestoreCancel(w http.ResponseWriter, _ *http.Request) {
	if err := os.Remove(store.StagedRestorePath(a.cfg.DBPath)); err != nil && !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "internal", "could not discard the staged restore")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// restoreStaged reports whether a restore is waiting for the next start.
func (a *app) restoreStaged() bool {
	_, err := os.Stat(store.StagedRestorePath(a.cfg.DBPath))
	return err == nil
}
