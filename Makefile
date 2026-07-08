# ssl-tournament-package — build
#
# Single self-contained binary with the web UI embedded (//go:embed).
# Modeled on ssl-game-controller. CGO is disabled everywhere so every target
# is a fully static binary that "just runs" — the whole point of this tier.

CMD     := ssl-tournament
PKG     := ./cmd/$(CMD)
VERSION ?= dev

# GOOS/GOARCH matrix (mirrors ssl-game-controller's release job).
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	linux/arm \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

export CGO_ENABLED := 0

# Stamp the version into the binary (main.version). -s -w strips debug info
# to shrink the static binaries.
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build run test clean release

# Native build → ./ssl-tournament
build:
	go build -ldflags "$(LDFLAGS)" -o $(CMD) $(PKG)

run:
	go run $(PKG)

test:
	go test ./...

# Cross-compile every target into ./release/<cmd>_<version>_<os>-<arch>[.exe]
release: clean
	@mkdir -p release
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=; [ "$$os" = "windows" ] && ext=.exe; \
		out=release/$(CMD)_$(VERSION)_$${os}-$${arch}$$ext; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out $(PKG) || exit 1; \
	done
	@echo "done → ./release"

clean:
	rm -rf release $(CMD) $(CMD).exe

all: test release
