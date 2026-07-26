#!/usr/bin/env bash
# Reeve agent installer/upgrader. Idempotent: re-run to upgrade in place.
#
#   curl -fsSL http://SERVER:8080/install.sh | sudo \
#     REEVE_SERVER_URL=http://SERVER:8080 REEVE_AGENT_TOKEN=TOKEN bash
#
# The server's SSH push-install runs this same script with the binary already
# copied in: REEVE_AGENT_BINARY points at it, REEVE_AGENT_SHA256 is the
# checksum to demand, and REEVE_UNINSTALL_SCRIPT a local copy of the
# uninstaller. No download happens in that mode.
#
set -euo pipefail

GITHUB_BASE="https://github.com/thehelvijs/Reeve/releases/download/edge"
# Public half of the release signing key. Agent binaries are signed with its
# private half; `minisign -V` here proves the download came from that key and not
# from whoever answered the request.
RELEASE_PUBKEY="RWS81m2QQ+FZ7EbZXYT8zFn36eqMxGdbE61cBTW7aMTELVKMDqc+z1nW"
BIN_PATH="/usr/local/bin/reeve-agent"
UNINSTALL_PATH="/usr/local/bin/reeve-agent-uninstall"
ENV_DIR="/etc/reeve-agent"
ENV_FILE="$ENV_DIR/agent.env"
UNIT_FILE="/etc/systemd/system/reeve-agent.service"
# Unsent pushes wait here rather than in /tmp, which any local user can
# pre-create as a symlink pointing somewhere this root process should not write.
BUFFER_DIR="/var/lib/reeve-agent/buffer"

uname_to_arch() {
  case "$1" in
    x86_64) echo "amd64" ;;
    aarch64 | arm64) echo "arm64" ;;
    armv7l) echo "armv7" ;;
    armv6l) echo "armv6" ;;
    i386 | i686) echo "386" ;;
    riscv64) echo "riscv64" ;;
    *) return 1 ;;
  esac
}

detect_arch() {
  local m
  m="$(uname -m)"
  if ! uname_to_arch "$m"; then
    echo "unsupported arch: $m (supported: amd64 arm64 armv7 armv6 386 riscv64)" >&2
    return 1
  fi
}

# Returns 0 (true) if VALUE contains a newline, carriage return, or other control char.
has_control_chars() {
  case "$1" in
    *[$'\001'-$'\037']*) return 0 ;;
    *) return 1 ;;
  esac
}

# fetch URL OUTFILE — download with curl or wget, whichever the host has.
fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    echo "need curl or wget to download $1" >&2
    return 1
  fi
}

# verify_signature FILE SIGFILE — check a minisign signature. Returns 2 when
# minisign is not installed, so the caller decides.
verify_signature() {
  if ! command -v minisign >/dev/null 2>&1; then
    return 2
  fi
  minisign -V -P "$RELEASE_PUBKEY" -x "$2" -m "$1" >/dev/null 2>&1
}

# install_minisign — try the host's package manager, quietly. The download path
# refuses to install an unverified binary, so this is the difference between a
# working install and a stop.
install_minisign() {
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq >/dev/null 2>&1 && apt-get install -y -qq minisign >/dev/null 2>&1
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y -q minisign >/dev/null 2>&1
  elif command -v apk >/dev/null 2>&1; then
    apk add --quiet minisign >/dev/null 2>&1
  elif command -v pacman >/dev/null 2>&1; then
    pacman -Sy --noconfirm --quiet minisign >/dev/null 2>&1
  else
    return 1
  fi
  command -v minisign >/dev/null 2>&1
}

# Library mode: let tests source helpers without executing the installer.
# `return` works when sourced; the `exit` is the executed-directly fallback.
if [ -n "${LV_LIB_ONLY:-}" ]; then
  # shellcheck disable=SC2317
  return 0 2>/dev/null || exit 0
fi

FORCE=0
SOURCE="${REEVE_INSTALL_SOURCE:-server}"
for arg in "$@"; do
  case "$arg" in
    --force) FORCE=1 ;;
    --github) SOURCE="github" ;;
    *) echo "unknown option: $arg" >&2; exit 1 ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "must run as root. Pipe to: sudo REEVE_SERVER_URL=... REEVE_AGENT_TOKEN=... bash" >&2
  exit 1
fi
if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemd (systemctl) required; this installer supports Linux hosts only" >&2
  exit 1
fi
if [ -z "${REEVE_SERVER_URL:-}" ] || [ -z "${REEVE_AGENT_TOKEN:-}" ]; then
  echo "REEVE_SERVER_URL and REEVE_AGENT_TOKEN are required" >&2
  exit 1
fi
if has_control_chars "$REEVE_SERVER_URL" || has_control_chars "$REEVE_AGENT_TOKEN"; then
  echo "REEVE_SERVER_URL and REEVE_AGENT_TOKEN must not contain newlines or control characters" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "warning: docker not found; container stats will be unavailable" >&2
fi

ARCH="$(detect_arch)"
# The server serves binaries under /dl/ and scripts at the root; GitHub Releases
# serve every asset from one flat base.
if [ "$SOURCE" = "github" ]; then
  DL_BASE="$GITHUB_BASE"
  SCRIPT_BASE="$GITHUB_BASE"
else
  DL_BASE="${REEVE_SERVER_URL%/}/dl"
  SCRIPT_BASE="${REEVE_SERVER_URL%/}"
fi
DL="$DL_BASE/agent-linux-$ARCH"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if [ -n "${REEVE_AGENT_BINARY:-}" ]; then
  # Pushed over SSH: the bytes arrived on an authenticated channel, so the
  # checksum is what guards against a local swap between copy and install.
  if [ ! -s "$REEVE_AGENT_BINARY" ]; then
    echo "REEVE_AGENT_BINARY=$REEVE_AGENT_BINARY is missing or empty" >&2
    exit 1
  fi
  cp "$REEVE_AGENT_BINARY" "$TMP/agent"
  if [ -n "${REEVE_AGENT_SHA256:-}" ]; then
    ACTUAL="$(sha256sum "$TMP/agent" | awk '{print $1}')"
    if [ "$REEVE_AGENT_SHA256" != "$ACTUAL" ]; then
      echo "checksum mismatch on the copied binary (expected $REEVE_AGENT_SHA256, got $ACTUAL)" >&2
      exit 1
    fi
    echo "checksum verified"
  fi
  chmod 0755 "$TMP/agent"
else
  echo "downloading $DL"
  fetch "$DL" "$TMP/agent"
  fetch "$DL.sha256" "$TMP/agent.sha256"
  # The .sha256 file is "<hex>  <name>"; verify against the downloaded binary.
  EXPECTED="$(awk '{print $1}' "$TMP/agent.sha256")"
  ACTUAL="$(sha256sum "$TMP/agent" | awk '{print $1}')"
  if [ "$EXPECTED" != "$ACTUAL" ]; then
    echo "checksum mismatch (expected $EXPECTED, got $ACTUAL)" >&2
    exit 1
  fi
  # The binary and its checksum come from the same place over the same
  # connection, so the checksum only proves the download was not corrupted.
  # The signature is what proves who built these bytes, and this script runs
  # them as root: it does not proceed without one.
  if ! fetch "$DL.minisig" "$TMP/agent.minisig" 2>/dev/null; then
    echo "no signature published for $DL; refusing to install an unverifiable binary as root." >&2
    echo "Rebuild the server with a release signing key, or set REEVE_ALLOW_UNVERIFIED=1 to override." >&2
    [ "${REEVE_ALLOW_UNVERIFIED:-}" = "1" ] || exit 1
    echo "REEVE_ALLOW_UNVERIFIED=1: installing on checksum alone" >&2
  else
    verify_signature "$TMP/agent" "$TMP/agent.minisig"
    case "$?" in
      0) echo "signature verified" ;;
      2)
        echo "minisign not installed; installing it to verify the download" >&2
        if install_minisign && verify_signature "$TMP/agent" "$TMP/agent.minisig"; then
          echo "signature verified"
        elif command -v minisign >/dev/null 2>&1; then
          echo "signature verification FAILED; refusing to install" >&2
          exit 1
        else
          echo "could not install minisign, so this download cannot be verified." >&2
          echo "Install minisign and re-run, or set REEVE_ALLOW_UNVERIFIED=1 to override." >&2
          [ "${REEVE_ALLOW_UNVERIFIED:-}" = "1" ] || exit 1
          echo "REEVE_ALLOW_UNVERIFIED=1: installing on checksum alone" >&2
        fi
        ;;
      *) echo "signature verification FAILED; refusing to install" >&2; exit 1 ;;
    esac
  fi
  chmod 0755 "$TMP/agent"
fi

TARGET_VERSION="$("$TMP/agent" --version)"

mkdir -p "$ENV_DIR"
umask 077
cat > "$ENV_FILE" <<EOF
REEVE_SERVER_URL=$REEVE_SERVER_URL
REEVE_AGENT_TOKEN=$REEVE_AGENT_TOKEN
REEVE_PUSH_INTERVAL=${REEVE_PUSH_INTERVAL:-15s}
EOF
chmod 0600 "$ENV_FILE"

mkdir -p "$BUFFER_DIR"
chmod 0700 "$BUFFER_DIR"

cat > "$UNIT_FILE" <<EOF
[Unit]
Description=Reeve agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
User=root
# Pinned so a writable directory earlier on an inherited PATH cannot decide
# which systemctl, docker or journalctl this root process runs.
Environment=PATH=/usr/sbin:/usr/bin:/sbin:/bin
EnvironmentFile=$ENV_FILE
ExecStart=$BIN_PATH
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

if [ -n "${REEVE_UNINSTALL_SCRIPT:-}" ] && [ -s "$REEVE_UNINSTALL_SCRIPT" ]; then
  cp "$REEVE_UNINSTALL_SCRIPT" "$TMP/uninstall.sh"
else
  fetch "$SCRIPT_BASE/uninstall.sh" "$TMP/uninstall.sh"
fi
if [ ! -s "$TMP/uninstall.sh" ] || ! head -n 1 "$TMP/uninstall.sh" | grep -q '^#!'; then
  echo "downloaded uninstall.sh failed sanity check (empty or missing shebang); refusing to install it" >&2
  exit 1
fi
install -m 0755 "$TMP/uninstall.sh" "$UNINSTALL_PATH"

if [ -e "$BIN_PATH" ]; then
  CURRENT_VERSION="$("$BIN_PATH" --version 2>/dev/null || echo unknown)"
  if [ "$CURRENT_VERSION" = "$TARGET_VERSION" ] && [ "$FORCE" -ne 1 ]; then
    echo "binary already current ($CURRENT_VERSION)"
  else
    echo "upgrading $CURRENT_VERSION -> $TARGET_VERSION"
    systemctl stop reeve-agent || true
    install -m 0755 "$TMP/agent" "$BIN_PATH"
  fi
else
  echo "installing $TARGET_VERSION"
  install -m 0755 "$TMP/agent" "$BIN_PATH"
fi

systemctl daemon-reload
systemctl enable --now reeve-agent
systemctl restart reeve-agent
echo "config reconciled, agent restarted. status:"
systemctl --no-pager --lines=0 status reeve-agent || true
