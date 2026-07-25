# Reeve

LAN-only, self-hosted control panel for a team's local and hosted tools. One Go
binary serves a web UI and a REST API on a single port; a per-host agent pushes
telemetry to it.

- **Catalog.** Every service the team runs, with owner, host, address and tags.
  Services are discovered from systemd, Docker and cron, or added by hand.
- **Monitoring.** CPU, memory, disk, network, load, temperature and GPU per
  host, plus service and container state, cron results and log errors.
  Thresholds fire alerts to webhooks and email.
- **Credentials.** Per-service secrets encrypted at rest, revealed only to the
  people and groups granted access, with every reveal audited.
- **Portal.** A public front door listing the tools marked public; everything
  else is behind sign-in with per-user and per-group visibility.

Nothing leaves the LAN. There is no hosted backend, no telemetry and no
external account.

## Run it

```sh
git clone https://github.com/thehelvijs/Reeve && cd Reeve
cp deploy/.env.example deploy/.env
# set REEVE_MASTER_KEY (openssl rand -base64 32) and
# REEVE_PUBLIC_URL=http://<lan-ip>:8080
docker compose -f deploy/docker-compose.yml up -d
```

That builds everything from the checkout — UI, embedded agents, server — and
needs nothing but Docker. No registry, no login, no toolchain. Add `--build`
after changing code; the `docker-compose.pull.yml` override in
[deploy/README.md](deploy/README.md) runs a published image instead.

The first account created becomes the admin. Add a host in the UI and it gives
you the one-line installer for that host's agent, or pushes the agent over SSH
for you.

Binaries, source builds, signing and backup are in
[deploy/README.md](deploy/README.md). The HTTP API is in
[contracts/API.md](contracts/API.md).

## Layout

```
server/     Go server: api, auth, rbac, catalog, crypto, ingest, alerts, jobs
agent/      Go agent: collectors + push loop, runs on monitored hosts
contracts/  push-payload and API types shared by server and agent
signing/    Ed25519 release signing and verification (minisign format)
web/        React + Vite + TS UI, built and embedded into the server binary
theme/      design tokens (theme.css, tailwind config)
deploy/     Dockerfiles, compose, install scripts
scripts/    release builder and the pre-commit gate
```

## License

```
Reeve - LAN-only control panel, credential store, and monitoring for a team's
self-hosted services.
Copyright (C) 2026  Helvijs Adams

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU Affero General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version. It is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
Affero General Public License in [LICENSE](LICENSE) for details.
```

You may run, modify, and self-host this freely, including inside a company. If
you modify it and offer it to others over a network, or ship it inside something
you distribute, the corresponding source has to be available under the same
license. The web UI links to this repository to satisfy that (AGPL section 13).

Contributions are accepted under the same license, signed off per the
[DCO](DCO) — see [CONTRIBUTING.md](CONTRIBUTING.md). Vulnerability reports go
to [SECURITY.md](SECURITY.md).
