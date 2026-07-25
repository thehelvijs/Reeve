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

Sealed with the same key: SMTP relay password, OAuth client secret, tool
descriptions, endpoints, and physical locations. Metric values and service names
stay plaintext, since they have to be searchable and aggregatable.

**Revealing a credential is audited.** Every reveal records who, which
credential, when, and the source IP. Grants and revocations are recorded too.

**Releases are signed.** Every agent binary ships with an Ed25519 signature in
minisign format, made from CI with a key held outside the repository. The agent
verifies that signature against a public key compiled into it *before* the
downloaded bytes are written to disk, and refuses any update it cannot verify. A
build with no key embedded disables self-update rather than falling back to
unsigned. `install.sh` verifies too when `minisign` is present on the host.

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
sending any credential, never stores the SSH credentials it is given, and mints a
fresh enrollment token for each push.

## Deployment expectations

This is designed for a LAN and ships no TLS of its own. If it is reachable from
anywhere untrusted:

- Put it behind a reverse proxy that terminates HTTPS, and set
  `REEVE_COOKIE_SECURE=true` so the session cookie is HTTPS-only.
- Close self-registration under **Settings → Sign-up**. It is open by default so
  a team can onboard itself, which is wrong for an exposed instance.
- Bind to a specific interface (`REEVE_ADDR`, or the compose port mapping)
  rather than `0.0.0.0`.

The agent runs as root, because reading systemd, Docker, cron, and journald
requires it. It only ever makes outbound connections to the server URL it was
configured with.

## Known limits

- No master-key rotation yet. Changing the key makes existing ciphertext
  unreadable.
- Single-tenant. Any admin can see every host, service, and audit record.
- The audit tables are append-only by convention, not cryptographically chained.
- Anyone with filesystem access to the database and the key material has
  everything. Protect the host accordingly.
