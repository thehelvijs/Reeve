# Security

## Reporting a vulnerability

Open a private security advisory through GitHub ("Security" tab → "Report a
vulnerability") rather than a public issue. Include what you did, what happened,
and the version from **Server → Details** in the UI or `GET /healthz`.

This is a small project with no paid support, so there is no formal response
window. Expect an acknowledgement within a few days.

## What this software protects, and how

**Credentials are encrypted before they touch disk.** AES-256-GCM, a random
96-bit nonce per secret, key from `REEVE_MASTER_KEY` (32 bytes, base64) in
the environment. The key is never written to the database or the repository, and
the server refuses to start without it. A stolen database alone reveals no
secret. A stolen database plus the key reveals all of them, so keep backups of
the two apart.

Sealed with the same key: the SMTP relay password and the OAuth client secret.
Everything else in the database is plaintext — tool names, descriptions,
endpoints, physical locations, host inventories, metric values and service
names — because it has to be searchable, sortable and aggregatable. Treat the
database as a full map of what the team runs and where; only the secrets
attached to those entries are encrypted.

**Revealing a credential is audited.** Every reveal records who, which
credential, which host it opens, when, and the source IP. Grants and revocations are recorded too.
The source IP is the peer address unless `REEVE_TRUST_PROXY=true` declares a
reverse proxy in front, in which case it is the left-most `X-Forwarded-For`
entry. Do not set that without a proxy that overwrites the header: the value
is otherwise whatever the caller typed, both here and in the login throttle.

**Writes must come from this origin.** A state-changing request carrying a
session cookie needs an `Origin` matching `REEVE_PUBLIC_URL`, the host it
arrived on, or an entry in `REEVE_ALLOWED_ORIGINS`. Token-authenticated callers
such as the agent's ingest push are unaffected. Responses carry a
content-security policy, `nosniff`, `X-Frame-Options: DENY` and a no-referrer
policy.

**Releases are signed.** Every agent binary ships with an Ed25519 signature in
minisign format, made from CI with a key held outside the repository. The agent
verifies that signature against a public key compiled into it *before* the
downloaded bytes are written to disk, and refuses any update it cannot verify. A
build with no key embedded disables self-update rather than falling back to
unsigned. `install.sh` installs `minisign` if the host lacks it and stops rather
than running an unverified binary as root; `REEVE_ALLOW_UNVERIFIED=1` overrides
that on a host where no package manager can supply it.

**Updates only move forward.** A signature proves who built a binary, not that
it is the newest they built. The signed trusted comment carries the version, and
the agent refuses an update that would move it to a lower one, so replaying a
genuine old release cannot walk a host back onto public bugs.

Verify a release yourself:

```sh
minisign -V -p agent/release_pubkey.txt -m agent-linux-amd64
```

**Login is throttled.** Five failures within fifteen minutes locks a key, both
per source IP and per account, for fifteen minutes. The deadline does not extend
on further attempts, so the lockout cannot be used to keep someone out.

**Password changes drop sessions.** Any password change, whether by the user, an
admin, a reset link, or the CLI, signs that account out everywhere.

**SSH push install** verifies the host key fingerprint an admin confirmed before
sending any credential, and mints a fresh enrollment token for each push. By
default it then keeps that login as a credential on the host — encrypted like
any other, revealable by admins and by the admin who ran the install, and
requestable by anyone else. Untick the box in the modal to install without
keeping it. A failed install stores nothing.

## Deployment expectations

This is designed for a LAN and ships no TLS of its own. If it is reachable from
anywhere untrusted:

- Put it behind a reverse proxy that terminates HTTPS, and set
  `REEVE_COOKIE_SECURE=true` so the session cookie is HTTPS-only.
- Close self-registration under **Settings → Sign-up**. It is open by default so
  a team can onboard itself, which is wrong for an exposed instance.
- The server can pace and pause a fleet-wide agent update, but it can never
  override a host that sets `REEVE_AUTO_UPDATE=false`. That veto is a
  deliberate trust boundary for a root process running on someone else's
  machine.
- Bind to a specific interface (`REEVE_ADDR`, or `REEVE_BIND` under compose)
  rather than `0.0.0.0`. Compose listens on every interface by default, so an
  instance is reachable from the LAN the moment it starts. Compose runs the
  server on the host network, so unlike a published container port — which
  Docker's NAT rules reach ahead of `ufw` — a host firewall rule does gate it:
  `ufw allow from 192.168.1.0/24 to any port 8080 proto tcp`.

The agent runs as root, because reading systemd, Docker, cron, and journald
requires it. It only ever makes outbound connections to the server URL it was
configured with. Its unit pins `PATH`, and unsent pushes buffer under
`/var/lib/reeve-agent` rather than `/tmp`, where any local user could
pre-create the path and steer what a root process writes and later replays.

## Known limits

- No master-key rotation yet. Changing the key makes existing ciphertext
  unreadable.
- Single-tenant. Any admin can see every host, service, and audit record.
- Any signed-in user, at any role, can list every host and its full
  systemd/Docker/cron inventory and running processes, see that a host holds
  credentials (labels and types, never the secret), read every account's email
  address, and mark a tool public on the anonymous portal. Visibility rules
  cover which *tools* someone sees and which *secrets* they can reveal, not the
  infrastructure those run on.
- **The agent can be told to act, not just report.** An admin can queue reboot,
  poweroff, and systemd or Docker start/stop/restart; the agent runs them as
  root. The action list is a fixed map in code, resolved to an argv with no
  shell anywhere in the path. The server refuses a target the host has not
  itself reported, and the agent independently refuses any target outside
  `[A-Za-z0-9_.@:-]{1,128}` — it does not assume its server is uncompromised.
  Control is on by default; `REEVE_ALLOW_CONTROL=false` on the machine overrides
  any server-side policy. Every command records the admin who asked, and that
  attribution is never reassigned when an account is deleted.
- A compromised server can therefore restart or power off every enrolled host.
  That was already true of a server pushing a malicious agent update, which is
  why releases are signed; the fixed action list bounds what this adds.
- Credentials belong to hosts, and a host has no owner but an admin. Adding,
  rotating, deleting and granting are therefore admin-only, and an admin can
  reveal every secret in the system.
- A per-account lockout is reachable by anyone who knows the address: five
  wrong passwords hold it for fifteen minutes. The deadline does not extend, so
  it cannot be held shut indefinitely, but it can be re-triggered.
- Restoring a backup replaces the whole database, password hashes included.
  It is admin-only and validated for shape, not for provenance.
- The audit tables are append-only by convention, not cryptographically chained.
- Anyone with filesystem access to the database and the key material has
  everything. Protect the host accordingly.
