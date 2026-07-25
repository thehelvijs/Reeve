# Contributing

## License and sign-off

This project is AGPL-3.0-or-later. Contributions are accepted under that same
license, and every commit needs a `Signed-off-by` line certifying the
[Developer Certificate of Origin](DCO):

```sh
git commit -s -m "fix: ..."
```

`-s` adds the trailer for you. There is no CLA, and copyright stays with you.
The sign-off is what records that you had the right to contribute the code.

## Before opening a pull request

The gate has to pass. Install it once so it runs on every commit:

```sh
ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
```

It runs `gofmt`, `go vet`, and `go test` across `server/`, `agent/`, and
`contracts/`, plus the web typecheck and lint when web files are staged. It must
stay under two seconds, so anything slower belongs in CI rather than the hook.

- **Every change ships with tests in the same commit.** A bug fix includes the
  test that would have caught it.
- **Frontend changes are reviewed in a browser**, not just by a passing
  typecheck. Screenshot the views you touched, and the ones you did not.
- Commits follow Conventional Commits (`feat:`, `fix:`, `refactor:`, `docs:`,
  `test:`, `chore:`, `perf:`, `build:`, `ci:`), one logical change each.
- Branch off `develop`. `main` is production and takes reviewed pull requests
  from `develop` only.

## Where things live

```
server/     Go server: api, auth, rbac, catalog, crypto, ingest, alerts, jobs
agent/      Go agent: collectors + push loop, runs as root on monitored hosts
contracts/  shared push-payload + API types imported by server and agent
signing/    Ed25519 release signing and verification (minisign format)
web/        React + Vite + TS UI, built and embedded into the server binary
deploy/     Dockerfiles, compose, install scripts
```

Two rules about that layout:

1. **`agent/` must not import `server/`.** The agent runs as root on other
   people's machines; its dependency surface stays as small as it is today.
2. **Protocol changes touch both sides in one commit.** Bumping
   `contracts.PushProtocolVersion` means updating producer, consumer, and tests
   together, since the server hard-rejects a mismatch at ingest.

## Design

User-facing UI follows [DESIGN.md](DESIGN.md). If a request and that document
conflict, ask rather than guessing.
