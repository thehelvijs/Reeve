# Project conventions

## Gate

Everything below has to pass before a commit. Install the hook once:

```sh
ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
```

```sh
make gate                    # go vet + go test across server, agent, contracts
make web-check               # tsc --noEmit + eslint --max-warnings 0
python3 -m pytest scripts/   # release builder + build config tests
```

`make web-build` stages the UI into `server/webdist`, `make server-assets`
stages install scripts and the embedded agent binaries into `server/`. Both are
needed before `make build`; the SSH-install and download handlers fail without
them. `make dist` produces the full release set.

## Layout rules

- `agent/` must not import `server/`. The agent runs as root on other people's
  machines; its dependency surface stays as small as it is today.
- Push-payload changes touch both sides in one commit: producer, consumer and
  tests together. There is no protocol version to negotiate. Reeve is
  pre-release with no agent in the world older than the server, so a shape
  change is a plain edit, not a compatibility problem.
- The schema is one file, `server/internal/store/schema.sql`, re-executed on
  every open. A schema change edits it in place; keep every statement
  idempotent (`IF NOT EXISTS`, `INSERT OR IGNORE`). There are no migrations,
  no version table and no data to migrate.
- Any change under `agent/` bumps the patch in `VERSION` in the same commit.
  The agent self-updates, so an operator reading the version has to be able to
  tell one deployed build from another.

## Implementation discipline

The working tree builds and passes tests at every commit.

## UI style

All user-facing UI follows `DESIGN.md`. New components match it; when it and a
request conflict, ask.

## Branches & environments

- `main` is the production branch and deploys to the production server.
- `develop` is the integration branch and deploys to the test server.
- Feature work happens on short-lived branches off `develop`. Completed tasks
  merge into `develop`, which deploys to test for review.
- Promoting to production is a reviewed pull request from `develop` into `main`.
- Never commit directly to `main`, and never merge to `main` without my
  explicit approval.
