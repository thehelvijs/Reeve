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
- The schema is one file, `server/internal/store/migrations/0001_init.sql`.
  Reeve is pre-release with no installed base, so a schema change edits that
  file in place. There are no numbered migrations and no data to migrate.

## Implementation discipline

Work proceeds one task at a time, committed separately. Commits are frequent
and small, one logical change each, and the working tree builds and passes
tests at every commit. Commit messages are short and imperative. Speculative
or "might need" features are left out.

Every change ships with tests in the same commit. A bug fix includes the test
that would have caught it.

## UI style

All user-facing UI follows `DESIGN.md` — a Linear-inspired system: dark-first
near-black canvas (never pure black), a single acid-lime accent (`#e4f222`)
used only for the brand, focus rings, and one primary action per view (with
dark text on the accent for contrast), hairline borders and a surface ladder
instead of drop shadows, Inter with `cv01`/`ss03` features and tight display
tracking, and a three-radius vocabulary (6 / 12 / 9999px). New components match
`DESIGN.md`; when it and a request conflict, ask.

Frontend changes are reviewed in a browser, not just by a passing typecheck.
Screenshot the views you touched, and the ones you did not.

## Branches & environments

- `main` is the production branch and deploys to the production server.
- `develop` is the integration branch and deploys to the test server.
- Feature work happens on short-lived branches off `develop`. Completed tasks
  merge into `develop`, which deploys to test for review.
- Promoting to production is a reviewed pull request from `develop` into `main`.
- Never commit directly to `main`, and never merge to `main` without my
  explicit approval.
