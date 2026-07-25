#!/usr/bin/env bash
# Reeve agent uninstaller. Local teardown only; does not touch the catalog.
set -euo pipefail

BIN_PATH="/usr/local/bin/reeve-agent"
UNINSTALL_PATH="/usr/local/bin/reeve-agent-uninstall"
ENV_DIR="/etc/reeve-agent"
UNIT_FILE="/etc/systemd/system/reeve-agent.service"

if [ "$(id -u)" -ne 0 ]; then
  echo "must run as root (sudo)" >&2
  exit 1
fi

systemctl disable --now reeve-agent 2>/dev/null || true
rm -f "$UNIT_FILE" "$BIN_PATH" "$UNINSTALL_PATH"
rm -rf "$ENV_DIR"
systemctl daemon-reload 2>/dev/null || true
echo "Reeve agent removed. Delete the host from the catalog UI if desired."
