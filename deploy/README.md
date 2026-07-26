# Deploying Reeve

LAN-only. Nothing here should be exposed to the public internet.

## Server (Docker Compose)

```sh
cp deploy/.env.example deploy/.env
# set REEVE_MASTER_KEY (openssl rand -base64 32) and
# REEVE_PUBLIC_URL=http://<lan-ip>:8080
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

The container runs on the host network, so that the server can reach machines
by their `.local` name: mDNS is multicast, and multicast out of a bridge network
stops at the bridge. It listens on `0.0.0.0:8080` by default; `REEVE_BIND` and
`REEVE_PORT` narrow that (`REEVE_BIND=192.168.1.10`, `REEVE_PORT=9000`). Docker
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

**LAN binding.** The compose file binds `0.0.0.0:8080` by default, so a fresh
instance is reachable from the network immediately — keep it behind your
firewall. Set `REEVE_BIND` to narrow it to one interface
(`REEVE_BIND=192.168.1.10`, or `127.0.0.1` when a reverse proxy on the same box
is the only thing that should reach it). Behind a proxy, also
set `REEVE_TRUST_PROXY=true` so the login throttle and the credential-reveal
audit see the real client address instead of the proxy's — and only then, since
the header is forgeable by anyone who can reach the server directly.

**Your firewall applies here.** A published container port (`-p 8080:8080`) is
reached through Docker's own NAT rules, which sit in front of `ufw` and ignore
it — a rule you thought was gating the port was not. On the host network the
server listens directly, so host rules do gate it. Scoping it to the LAN, which
is who this is for:

```sh
sudo ufw allow from 192.168.1.0/24 to any port 8080 proto tcp
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

On the host row, **Install over SSH**. Give it the machine's address, an SSH
user, and a password or private key (plus a sudo password unless the user is
root or has NOPASSWD). The server shows the host key fingerprint for you to
confirm, then copies the matching agent build and the installer over and runs
it. Credentials are used for that one operation and never stored, and a fresh
enrollment token is minted for each push. **Remove agent** does the reverse and
leaves the host in the catalog with its history.

This needs `REEVE_PUBLIC_URL` set, since it is the address the agent is
told to push to.

The address is resolved by the server, not your browser. A `.local` name is
answered by multicast, which no DNS server and no static Go binary can do, so
the server sends the mDNS query itself when DNS comes up empty. That needs the
container on the host network, which the compose file does by default. Under a
bridge network — or Docker Desktop, whose host is a VM — multicast never reaches
the LAN: use the machine's IP, or map the name with
`extra_hosts: ["somehost.local:192.168.1.20"]`.

### Pulled by the host with curl

```sh
curl -fsSL http://<server>:8080/install.sh | sudo \
  REEVE_SERVER_URL=http://<server>:8080 \
  REEVE_AGENT_TOKEN=<token> bash
```

Installs a root systemd unit with full systemd/cron/journald/docker visibility.
**Re-run the same command to upgrade** in place (the token is preserved).

Pull from GitHub Releases instead of the server (bootstrap / server
unreachable): append `--github` or set `REEVE_INSTALL_SOURCE=github`.

**Auto-update.** By default the server paces the fleet's self-update rollout
and tells each agent when to check (Hosts page, admin-only: pause, resume,
force a single host). `REEVE_AUTO_UPDATE=false` is a local veto the server
can never override: the agent refuses every update, including one the server
explicitly asks for. `REEVE_UPDATE_INTERVAL` (a Go duration, default `1h`) no
longer drives routine updates; it now only paces the recovery check that fires
when no valid server ack has arrived in `2 × REEVE_UPDATE_INTERVAL`, so an
agent that falls out of contact with the server still updates itself
unattended.

**Release signing.** Agent binaries are signed with the project's Ed25519
release key. The agent verifies that signature before installing any self-update
and refuses an update it cannot verify. `install.sh` checks it too when
`minisign` is present on the host (`apt install minisign`), and warns when it is
not. Maintainers: create the key once with
`go run ./scripts/sign -genkey -out ~/.reeve/release-key`, commit the
printed public key to `agent/release_pubkey.txt`, and add the secret key to CI
as the `REEVE_SIGNING_KEY` secret. Without that secret CI still builds, but
the artifacts go out unsigned and deployed agents will not self-update to them.

### Uninstall

```sh
curl -fsSL http://<server>:8080/uninstall.sh | sudo bash
# or, offline on the host:
sudo reeve-agent-uninstall
```

Local teardown only; remove the host from the catalog in the UI separately.
**Remove agent** on the host row does the same thing over SSH.

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
