#!/usr/bin/env bash
# Gate lane: fast, deterministic checks that must pass before every commit.
set -euo pipefail

# Go packages, excluding web/node_modules which can contain stray .go files.
GO_PKGS="./server/... ./agent/... ./contracts/..."

# Go: format check, vet, tests.
if ! gofmt -l server agent contracts | (! grep .); then
  echo "gofmt: files need formatting (run: gofmt -w server agent contracts)" >&2
  exit 1
fi
go vet $GO_PKGS
go test $GO_PKGS

# Web: only when web files are staged (keeps the hook fast otherwise).
if git diff --cached --name-only | grep -q '^web/'; then
  npm --prefix web run typecheck
  npm --prefix web run lint
fi
