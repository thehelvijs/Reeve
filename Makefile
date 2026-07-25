.PHONY: test vet fmt gate web-build web-check build server-assets dist image

# Go packages, excluding web/node_modules which can contain stray .go files.
GO_PKGS := ./server/... ./agent/... ./contracts/...

# Gate lane: what the pre-commit hook runs.
gate: vet test

test:
	go test $(GO_PKGS)

vet:
	go vet $(GO_PKGS)

fmt:
	gofmt -w server agent contracts

# Build the UI and stage it for embedding into the server binary.
web-build:
	npm --prefix web run build
	rm -rf server/webdist
	cp -r web/dist server/webdist

web-check:
	npm --prefix web run typecheck
	npm --prefix web run lint

# Stage host scripts + embedded agent binaries into the server package.
server-assets:
	mkdir -p server/scripts
	cp deploy/install.sh deploy/uninstall.sh server/scripts/
	python3 scripts/release.py --embed

VERSION := $(shell cat VERSION)
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Build both binaries into bin/ (run web-build first for the embedded UI).
build:
	go build -ldflags "$(LDFLAGS)" -o bin/server ./server
	go build -ldflags "$(LDFLAGS)" -o bin/agent ./agent

# Full release artifacts in dist/: every agent arch plus server binaries with
# the UI and agents embedded. What CI runs on a tag.
dist: web-build server-assets
	python3 scripts/release.py --release --server --version $(VERSION)

# Build the server image the compose file pulls, tagged for local use.
image:
	docker build -f deploy/Dockerfile.server -t reeve:$(VERSION) --build-arg VERSION=$(VERSION) .
