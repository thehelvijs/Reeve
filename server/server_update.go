package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"
)

const settingServerUpdateChannel = "server_update.channel"

// The build this instance last started as, and when it changed to it. There is
// nothing else to ask: the updater replaces the container and the new binary is
// the only witness that anything happened.
const (
	settingServerUpdateVersion = "server_update.version"
	settingServerUpdateAt      = "server_update.updated_at"
)

// recordServerVersion notes a version change at startup, which is what "last
// updated" reports. A restart on the same build is not an update and must not
// move the stamp, or every reboot would read as one.
func (a *app) recordServerVersion() {
	if a.cfg.Version == "" {
		return
	}
	last, ok := a.db.GetSetting(settingServerUpdateVersion)
	if ok && last == a.cfg.Version {
		return
	}
	if err := a.db.SetSetting(settingServerUpdateVersion, a.cfg.Version); err != nil {
		log.Printf("server update: could not record the running version: %v", err)
		return
	}
	// A first start has nothing to have updated from, so it records the version
	// without claiming an update happened.
	if !ok {
		return
	}
	if err := a.db.SetSetting(settingServerUpdateAt, time.Now().UTC().Format(time.RFC3339)); err != nil {
		log.Printf("server update: could not record the update time: %v", err)
	}
}

const (
	channelRelease = "release"
	channelDevelop = "develop"
	channelMain    = "main"
)

// channelTags maps a channel to the image tag CI publishes it under: `release`
// moves only on a tagged release, the branch channels on every push to them.
//
// The updater sidecar reads the tag out of the data directory rather than being
// told a channel name, so this map is the only place the two agree. It also
// bounds what an admin can select: the registry and repository are fixed in the
// compose file, and a channel can only ever choose among these three tags.
var channelTags = map[string]string{
	channelRelease: "latest",
	channelDevelop: "develop",
	channelMain:    "main",
}

// channelFileName is read by the updater on every poll, so a switch takes
// effect on the next one with nothing to restart.
const channelFileName = "update-channel"

// serverUpdateChannel is the channel this instance follows, defaulting to
// release: an instance nobody has configured should not track a branch.
func (a *app) serverUpdateChannel() string {
	v, ok := a.db.GetSetting(settingServerUpdateChannel)
	if !ok || channelTags[v] == "" {
		return channelRelease
	}
	return v
}

// serverUpdateView is what the settings page reads: the channel it follows, the
// build it is on, and when that build arrived.
func (a *app) serverUpdateView() serverUpdateView {
	at, _ := a.db.GetSetting(settingServerUpdateAt)
	return serverUpdateView{
		Channel:   a.serverUpdateChannel(),
		Version:   a.cfg.Version,
		UpdatedAt: at,
	}
}

// channelFilePath puts the file beside the database, which is the one directory
// an operator already has to persist, so the updater's mount follows the
// server's own volume rather than a second path to keep in step.
func (a *app) channelFilePath() string {
	return filepath.Join(filepath.Dir(a.cfg.DBPath), channelFileName)
}

// writeChannelFile publishes the selected channel's tag for the updater,
// atomically: the updater reads this file on a timer and a half-written tag
// would either name no image or, worse, name a different one.
func (a *app) writeChannelFile(channel string) error {
	tag, ok := channelTags[channel]
	if !ok {
		return errors.New("unknown update channel: " + channel)
	}
	path := a.channelFilePath()
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+channelFileName+"-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.WriteString(tag + "\n")
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(tmpPath)
		return errors.Join(writeErr, closeErr)
	}
	// The updater's mount is read-only, but the file is world-readable so the
	// sidecar does not have to run as the same user the server does.
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// syncChannelFile restates the stored channel on startup. The database is the
// record; this file is a projection of it, and one that a restored backup or a
// fresh volume would otherwise leave behind, silently parking the instance on
// release while the UI reported a branch.
func (a *app) syncChannelFile() {
	if err := a.writeChannelFile(a.serverUpdateChannel()); err != nil {
		log.Printf("server update: could not publish update channel: %v", err)
	}
}
