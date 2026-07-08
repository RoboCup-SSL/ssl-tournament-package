#!/usr/bin/env sh
# Install ssl-tournament as a systemd service (headless / always-on Linux).
#
# Downloads the release binary for this machine's architecture, creates a
# locked-down system user, installs a systemd unit, and starts it on boot.
#
# Usage (as root):
#   ./install.sh                 # install the latest release
#   VERSION=v0.1.0 ./install.sh  # install a specific tag
#   PORT=9000 ./install.sh       # serve on a different port
#
# While the repo is PRIVATE, release assets require auth — export a token
# with 'repo' scope:
#   GITHUB_TOKEN=ghp_xxx ./install.sh
# Once the repo is public, no token is needed.
set -eu

REPO="RoboCup-SSL/ssl-tournament-package"
BIN_NAME="ssl-tournament"
INSTALL_DIR="/usr/local/bin"
SVC_USER="ssl-tournament"
UNIT_NAME="ssl-tournament.service"
UNIT_PATH="/etc/systemd/system/${UNIT_NAME}"
VERSION="${VERSION:-latest}"
PORT="${PORT:-8080}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

die() { echo "error: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root (try: sudo $0)"
[ "$(uname -s)" = "Linux" ] || die "headless install is Linux/systemd only (use the desktop build on macOS/Windows)"
command -v systemctl >/dev/null 2>&1 || die "systemd (systemctl) not found"
command -v curl >/dev/null 2>&1 || die "curl is required"

# Map machine arch → the GOARCH used in release asset names.
case "$(uname -m)" in
	x86_64|amd64)        ARCH=amd64 ;;
	aarch64|arm64)       ARCH=arm64 ;;
	armv7l|armv6l|arm)   ARCH=arm ;;
	*) die "unsupported architecture: $(uname -m)" ;;
esac

# Auth header for private-repo downloads (empty once the repo is public).
AUTH=""
[ -n "${GITHUB_TOKEN:-}" ] && AUTH="Authorization: Bearer ${GITHUB_TOKEN}"

# Resolve the concrete tag: assets are named with the version, so even for
# "latest" we first look up the tag via the API.
if [ "$VERSION" = "latest" ]; then
	echo "resolving latest release tag..."
	VERSION=$(curl -fsSL ${AUTH:+-H "$AUTH"} "https://api.github.com/repos/${REPO}/releases/latest" \
		| grep -m1 '"tag_name"' | cut -d'"' -f4)
	[ -n "$VERSION" ] || die "could not resolve latest release (private repo? set GITHUB_TOKEN)"
fi

ASSET="${BIN_NAME}_${VERSION}_linux-${ARCH}"
echo "installing ${BIN_NAME} ${VERSION} (linux-${ARCH})"

# Download the binary. For private repos the browser URL 404s, so hit the API
# asset endpoint with Accept: octet-stream, which honors the token.
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
if [ -n "$AUTH" ]; then
	ASSET_URL=$(curl -fsSL -H "$AUTH" "https://api.github.com/repos/${REPO}/releases/tags/${VERSION}" \
		| grep -o "\"url\": \"[^\"]*releases/assets/[0-9]*\"" | cut -d'"' -f4 \
		| while read -r u; do
			name=$(curl -fsSL -H "$AUTH" "$u" | grep -m1 '"name"' | cut -d'"' -f4)
			[ "$name" = "$ASSET" ] && echo "$u" && break
		done)
	[ -n "$ASSET_URL" ] || die "asset ${ASSET} not found in release ${VERSION}"
	curl -fSL -H "$AUTH" -H "Accept: application/octet-stream" -o "$TMP" "$ASSET_URL"
else
	curl -fSL -o "$TMP" "https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
fi

# Create the locked-down service user if missing.
if ! id "$SVC_USER" >/dev/null 2>&1; then
	echo "creating system user ${SVC_USER}"
	useradd --system --no-create-home --shell /usr/sbin/nologin "$SVC_USER"
fi

# Install the binary.
install -m 0755 "$TMP" "${INSTALL_DIR}/${BIN_NAME}"
echo "installed ${INSTALL_DIR}/${BIN_NAME} ($(${INSTALL_DIR}/${BIN_NAME} --version))"

# Install the systemd unit (from the copy shipped alongside this script),
# rewriting the port if the caller asked for a non-default one.
sed "s/--port 8080/--port ${PORT}/" \
	"${SCRIPT_DIR}/systemd/${UNIT_NAME}" > "$UNIT_PATH"

systemctl daemon-reload
systemctl enable --now "$UNIT_NAME"

echo
echo "done. ssl-tournament is running and will start on boot."
echo "  status:  systemctl status ${UNIT_NAME}"
echo "  logs:    journalctl -u ${UNIT_NAME} -f"
echo "  open:    http://$(hostname -I 2>/dev/null | awk '{print $1}'):${PORT}"
