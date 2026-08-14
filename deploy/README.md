# Deploying Reeve

LAN-only. Nothing here should be exposed to the public internet.

## Server (Docker Compose)

```sh
cp deploy/.env.example deploy/.env
# set REEVE_MASTER_KEY (openssl rand -base64 32)
docker compose -f deploy/docker-compose.yml up -d
```

`.env` lives beside the compose file, not at the repo root: Compose reads it
from the compose file's directory.

That builds the image from this checkout and tags it `reeve:source`. Nothing is
pulled from a registry, so no login, no published package and no network access
to ghcr is involved. The build needs only Docker: `deploy/Dockerfile.server`
installs the web dependencies, builds the UI, embeds the agent binaries and
compiles the server itself, so no local Go, Node or Python toolchain is
required.

One container serves everything: the web UI is built and embedded into the server
binary, so the SPA, the REST API, the agent installer, and the agent binaries all
come off the same port. There is no separate frontend container or dev server to
run.

`up -d` also starts an **agent for the machine Reeve runs on**, because that
machine is a host like any other and usually the one already running something
worth watching. It enrols itself: the server writes an enrollment token for its
own host row into the data volume and the agent reads it from there, so there is
nothing to paste and no secret in `.env`. It appears as **Reeve server** in
Hosts, with its containers and metrics like any other machine.

That agent is part of the deploy, not of the fleet rollout — it self-updates by
being repulled or rebuilt with the rest of the stack, so it runs with
`REEVE_AUTO_UPDATE=false` and reads "updates off". Running it in a container
means Docker visibility but not the host's systemd or journal; for those, install
the agent on that machine the ordinary way (**Install command** on its host page)
and it takes over the row — the server stops sampling itself as soon as an agent
reports. Don't want it at all? `docker compose -f deploy/docker-compose.yml up -d
server` starts the server alone.

`up -d` reuses the existing `reeve:source` image. After changing code, rebuild:

```sh
docker compose -f deploy/docker-compose.yml up -d --build
```

To run a published image instead of building, add the pull override. The
package is private, so log in first; `REEVE_IMAGE` picks the tag and should be
pinned to a version in production:

```sh
docker login ghcr.io
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.pull.yml up -d
```

**Server auto-update.** That override also starts an `updater` container, so an
instance installed this way stays current with nothing else to run. Every
`REEVE_UPDATE_POLL_SECS` (default hourly) it re-runs `up -d server` with the
current channel's tag: nothing happens when neither the digest nor the channel
moved, which is the ordinary case. The web UI is embedded in the server binary,
so one pull updates both, and `schema.sql` is re-executed on every open, so
there is nothing to migrate across a restart. Agents follow on their own
afterwards through the paced rollout below.

**Channels.** **Settings → Server updates**, admin-only, picks which stream of
builds this instance follows:

| Channel | Image tag | Moves on |
| --- | --- | --- |
| `release` (default) | `:latest` | a tagged `v*` release |
| `develop` | `:develop` | every push to `develop` |
| `main` | `:main` | every push to `main` |

The server writes the resolved tag to `update-channel` in the data volume and
the updater picks it up on the next poll, so switching is a click and a wait,
not a redeploy. Switching *down* a channel — develop back to release — runs an
older binary against a database a newer build has already opened; the UI says so
before you press the button. The tag list lives in two places by necessity, the
`channelTags` map in `server/server_update.go` and the updater's own whitelist
here; a test fails if they drift.

Holding an instance still: pin `REEVE_IMAGE` to a version *and* leave the
channel alone, or drop the `updater` service. There is deliberately no
"arbitrary image" field in the UI — an admin session can choose among three
published tags of one repository, and the repository itself
(`REEVE_IMAGE_REPO`) is set here, not in the database.

The updater holds the Docker socket, which is root on that host: nothing inside
the server container can replace the container it is running in, so this is what
the capability costs. It mounts the data volume read-only and the compose files
read-only. While the ghcr package is private it cannot authenticate on its own —
mount host credentials into it (`~/.docker/config.json:/config.json:ro`) or make
the package public, otherwise its pulls fail, it logs and the server stays on
the build it is running.

Building from this checkout instead? There is no image to pull, so nothing
updates itself; `up -d --build` is the update, and the channel selector has
nothing behind it. Same for the raw `server-linux-*` binaries — supervise them
yourself, and note they ship unsigned, unlike the agent builds.

The container runs on the host network, so that the server can reach machines
by their `.local` name: mDNS is multicast, and multicast out of a bridge network
stops at the bridge. It listens on `0.0.0.0:7338` by default; `REEVE_BIND` and
`REEVE_PORT` narrow that (`REEVE_BIND=192.168.1.10`, `REEVE_PORT=7400`). Docker
Desktop is the exception — its "host" is a VM, not your machine, so `.local`
names will not resolve there. The image carries its own `HEALTHCHECK` — the
binary probes its own `/healthz`, so `docker ps` shows `healthy` without curl in
the image.

Not using Docker? Each release also publishes `server-linux-amd64` and
`server-linux-arm64` with the UI and agent binaries already embedded. Run one
with `REEVE_MASTER_KEY` set; it needs no other files.

The first account created becomes the admin. SQLite lives in the `reeve-data`
volume. Sign-up is open by default so colleagues can register themselves —
close it under **Settings → Sign-up** once the team is in.

**Account management from a shell.** `server users` administers accounts without
the web UI — for a lost admin password, an instance with no admin left, or a
scripted setup. It needs no master key, and works while the server is running.

```sh
server users list
server users create you@example.com -password 'pw' -role admin -name "Your Name"
server users set-password you@example.com -password 'new-pw'
server users set-role dev@example.com admin        # or basic
server users set-name dev@example.com "Dev Name"
server users enable  dev@example.com
server users disable dev@example.com
server users delete  dev@example.com [-reassign-to ops@example.com] -yes
```

In Docker, prefix with `docker compose -f deploy/docker-compose.yml exec server`:

```sh
docker compose -f deploy/docker-compose.yml exec server \
  /usr/local/bin/server users create you@example.com -password 'pw' -role admin
```

It refuses anything that would leave the instance unadministerable: demoting,
disabling, or deleting the last active admin. `delete` needs `-yes` and hands the
account's tools and credentials to another admin (or to `-reassign-to`).
Setting a password signs that account out everywhere.
`server -reset-password <email> -password <pw>` remains as a shorthand for
`server users set-password`.

**Backups.** **Server → Backup and restore** downloads a consistent copy of the
database and stages an uploaded one (applied on the next restart). Credential
ciphertext travels inside it; the master key does not, so store the key
separately or the backup is unreadable.

**LAN binding.** The compose file binds `0.0.0.0:7338` by default, so a fresh
instance is reachable from the network immediately — keep it behind your
firewall. Set `REEVE_BIND` to narrow it to one interface
(`REEVE_BIND=192.168.1.10`, or `127.0.0.1` when a reverse proxy on the same box
is the only thing that should reach it). Behind a proxy, also
set `REEVE_TRUST_PROXY=true` so the login throttle and the credential-reveal
audit see the real client address instead of the proxy's — and only then, since
the header is forgeable by anyone who can reach the server directly.

**Your firewall applies here.** A published container port (`-p 7338:7338`) is
reached through Docker's own NAT rules, which sit in front of `ufw` and ignore
it — a rule you thought was gating the port was not. On the host network the
server listens directly, so host rules do gate it. Scoping it to the LAN, which
is who this is for:

```sh
sudo ufw allow from 192.168.1.0/24 to any port 7338 proto tcp
```

Anything arriving from outside that range is then dropped, including containers
on the Docker bridges. Nothing in Reeve needs that path — the agents connect
outbound to the server, never the reverse.

## Coolify

1. New Resource → Docker Compose → point at this repo, compose path
   `deploy/docker-compose.yml`.
2. Set the `REEVE_MASTER_KEY` (and `REEVE_PUBLIC_URL`) environment
   variables in Coolify's UI — do not commit them.
3. Attach a persistent volume for `/data`.
4. Restrict the exposed domain/port to the LAN.

Coolify routes to containers over its own network, so drop `network_mode: host`
there — at the cost of installing agents by IP rather than by `.local` name.

## Agent

Enroll a host in the UI (**Hosts → Add host**). Two ways to get the agent onto
the machine; both run the same installer.

### Pushed from the server over SSH

On the host row, **Install over SSH** — shown on any host that is not already
online and on the published build, since a machine that is fine needs nothing
done to it. Give it the machine's address, an SSH
user, and a password or private key (plus a sudo password unless the user is
root or has NOPASSWD). The server shows the host key fingerprint for you to
confirm, then copies the matching agent build and the installer over and runs
it. A fresh enrollment token is minted for each push. The login is kept as a
credential on that host so the next person does not have to hunt for it —
untick "Save this login as a credential" to skip that; an uninstall never
stores anything. **Remove agent over SSH** does the reverse and leaves the host
in the catalog with its history; it lives on the host's own page, next to
Delete, rather than in the list — both end a host, and neither belongs one click
away in a row you are scrolling past.

The address the agent is told to push to defaults to whatever host you reached
the UI on, so opening it at `http://<lan-ip>:7338` needs no configuration. Set
`REEVE_PUBLIC_URL` when that differs from the address agents should use, or when
you are working over `localhost` — a loopback origin is refused, since an agent
on another machine cannot dial it.

The address is resolved by the server, not your browser. A `.local` name is
answered by multicast, which no DNS server and no static Go binary can do, so
the server sends the mDNS query itself when DNS comes up empty. That needs the
container on the host network, which the compose file does by default. Under a
bridge network — or Docker Desktop, whose host is a VM — multicast never reaches
the LAN: use the machine's IP, or map the name with
`extra_hosts: ["somehost.local:192.168.1.20"]`.

### Pulled by the host with curl

```sh
curl -fsSL http://<server>:7338/install.sh | sudo \
  REEVE_SERVER_URL=http://<server>:7338 \
  REEVE_AGENT_TOKEN=<token> bash
```

Installs a root systemd unit with full systemd/cron/journald/docker visibility.
**Re-run the same command to upgrade** in place (the token is preserved).

Pull from GitHub Releases instead of the server (bootstrap / server
unreachable): append `--github` or set `REEVE_INSTALL_SOURCE=github`.

**Auto-update.** By default the server paces the fleet's self-update rollout
and tells each agent when to check (Hosts page, admin-only: pause, resume,
force a single host — **Update** appears on any online row that is not on the
published build, including one whose auto-update is off, since "off" means "not
on the paced rollout" rather than "never". A forced update sits outside the
rollout: it ignores the cap and cannot pause anyone. The one exception is a host
that vetoes locally with `REEVE_AUTO_UPDATE=false`, which no override reaches
because the agent ignores the ack). Current means "running the binary this server publishes":
the agent reports the sha256 of its own binary and the server compares it to
the build it serves, the same comparison the agent's self-update makes. Version
strings decide nothing, so a rebuild at the same version still rolls out. A host
that reports no checksum reads as **build unknown** and is never told to update;
re-running the install command above fixes that. `REEVE_AUTO_UPDATE=false` is a local veto the server
can never override: the agent refuses every update, including one the server
explicitly asks for. `REEVE_UPDATE_INTERVAL` (a Go duration, default `1h`) no
longer drives routine updates; it now only paces the recovery check that fires
when no valid server ack has arrived in `2 × REEVE_UPDATE_INTERVAL`, so an
agent that falls out of contact with the server still updates itself
unattended.

**Credentials from an install.** An SSH push-install keeps the login it just
proved works as a credential on that host, encrypted at rest, with a standing
grant for the admin who ran it. Untick "Save this login as a credential" in the
modal to skip it. Nothing is kept when the install fails or when there was no
secret (forwarded key, NOPASSWD sudo).

**Filesystems.** Each push reports every mounted filesystem worth measuring —
mount point, device, type, used and total — listed on the host page fullest
first, since the disk about to run out is rarely the root one. Kernel and
virtual mounts are dropped, snap loop devices with them, and a bind mount counts
once per device. The Server page shows the same for the machine running Reeve.
The disk chart and the disk alert threshold still track the root filesystem.

**Remote control.** An admin can queue a fixed set of actions for a host from
its page — restart, shut down, and start/stop/restart of a systemd unit or a
Docker container — which the agent collects on its next push (~15s) and runs as
root. On by default. `REEVE_ALLOW_CONTROL=false` on the machine is a local veto
the server cannot override: the agent ignores every command and reports that it
will, and the buttons go dead with that reason on screen. There is no
"run any command" action, by design.

**Release signing.** Agent binaries are signed with the project's Ed25519
release key. The agent verifies that signature before installing any self-update
and refuses an update it cannot verify. `install.sh` checks it too when
`minisign` is present on the host (`apt install minisign`), and warns when it is
not. Maintainers: create the key once with
`go run ./scripts/sign -genkey -out ~/.reeve/release-key`, commit the
printed public key to `agent/release_pubkey.txt`, and add the secret key to CI
as the `REEVE_SIGNING_KEY` secret. Without that secret CI still builds, but
the artifacts go out unsigned and deployed agents will not self-update to them.

The same applies to a local `up -d --build`: put the key's one line in
`deploy/.env` as `REEVE_SIGNING_KEY` and compose passes it to the build as a
secret. It reaches the build only, never the running container or the image.
With it empty or missing, the build says so and the embedded agents are
unsigned.

### Uninstall

```sh
curl -fsSL http://<server>:7338/uninstall.sh | sudo bash
# or, offline on the host:
sudo reeve-agent-uninstall
```

Local teardown only; remove the host from the catalog in the UI separately.
**Remove agent over SSH**, on the host's page, does the same thing remotely.

### Docker (alternative)

The UI also shows a `docker run` command. It sees Docker via the mounted socket
but not host systemd/cron/journald.

## Master-key custody (read this)

`REEVE_MASTER_KEY` encrypts every stored credential. It is **not** kept in
the database or the repo.

- Losing it means every stored credential is unrecoverable. Back it up somewhere
  safe and separate from the DB backup.
- Anyone with both the key and the database can decrypt all credentials — treat
  the key as a top secret and the DB volume as sensitive.
- The server refuses to start without it.
