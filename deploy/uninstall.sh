#!/usr/bin/env sh
# Uninstall the ssl-tournament systemd service.
#
# By design the data dir (/var/lib/ssl-tournament, holding the tournament DB)
# is KEPT so a tournament isn't lost. Pass --purge to remove it too.
#
# Usage (as root):
#   ./uninstall.sh            # remove service + binary, keep data
#   ./uninstall.sh --purge    # also delete /var/lib/ssl-tournament and the user
set -eu

BIN_NAME="ssl-tournament"
INSTALL_DIR="/usr/local/bin"
SVC_USER="ssl-tournament"
UNIT_NAME="ssl-tournament.service"
UNIT_PATH="/etc/systemd/system/${UNIT_NAME}"
DATA_DIR="/var/lib/ssl-tournament"

PURGE=0
[ "${1:-}" = "--purge" ] && PURGE=1

die() { echo "error: $*" >&2; exit 1; }
[ "$(id -u)" -eq 0 ] || die "must run as root (try: sudo $0)"

# Stop & disable the service if present.
if systemctl list-unit-files "$UNIT_NAME" >/dev/null 2>&1; then
	systemctl disable --now "$UNIT_NAME" 2>/dev/null || true
fi
rm -f "$UNIT_PATH"
systemctl daemon-reload

rm -f "${INSTALL_DIR}/${BIN_NAME}"

if [ "$PURGE" -eq 1 ]; then
	echo "purging data dir and service user"
	rm -rf "$DATA_DIR"
	id "$SVC_USER" >/dev/null 2>&1 && userdel "$SVC_USER" 2>/dev/null || true
	echo "done. everything removed."
else
	echo "done. service and binary removed."
	echo "data kept at ${DATA_DIR} (re-run with --purge to delete it)."
fi
