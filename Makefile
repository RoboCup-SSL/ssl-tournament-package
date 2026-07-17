# ssl-tournament-package — build
#
# Two binaries, both static (CGO disabled) with the web UI embedded (//go:embed):
#   ssl-tournament          — the server. Cross-platform (desktop/manual/`make run`).
#   ssl-tournament-service  — Linux-only headless artifact; run it to install
#                             itself as a systemd service.
# Modeled on ssl-game-controller's multi-cmd + release layout.

SERVER  := ssl-tournament
SERVICE := ssl-tournament-service
VERSION ?= dev

# ssl-tournament ships for every desktop/server OS.
SERVER_PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	linux/arm \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

# The systemd installer only makes sense on Linux.
SERVICE_PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	linux/arm

export CGO_ENABLED := 0

# Stamp the version into main.version. -s -w strips debug info to shrink binaries.
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build run test clean release frontend

# Native build of the server → ./ssl-tournament
build:
	go build -ldflags "$(LDFLAGS)" -o $(SERVER) ./cmd/$(SERVER)

# Rebuild the embedded web UI into frontend/dist (committed). Requires Node.
# Run after changing frontend/ source, then commit the regenerated dist.
frontend:
	cd frontend && npm ci && npm run build

run:
	go run ./cmd/$(SERVER)

test:
	go test ./...

# Cross-compile both binaries into ./release/<cmd>_<version>_<os>-<arch>[.exe]
release: clean
	@mkdir -p release
	@for p in $(SERVER_PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=; [ "$$os" = "windows" ] && ext=.exe; \
		out=release/$(SERVER)_$(VERSION)_$${os}-$${arch}$$ext; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out ./cmd/$(SERVER) || exit 1; \
	done
	@for p in $(SERVICE_PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=release/$(SERVICE)_$(VERSION)_$${os}-$${arch}; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out ./cmd/$(SERVICE) || exit 1; \
	done
	@echo "done → ./release"

clean:
	rm -rf release $(SERVER) $(SERVER).exe $(SERVICE)

all: test release
