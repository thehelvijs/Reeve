#!/usr/bin/env bash
# Live install/uninstall of the agent on real hardware, over SSH, through the
# server's own API — the one path the Go suite cannot cover, because it fakes
# the SSH host in-process.
#
# Opt-in and never part of `make gate`: it needs a reachable machine and a
# credential. Nothing here runs unless you pass a target.
#
#   scripts/live_ssh_install_test.sh --server http://192.168.1.120:8081 \
#     --admin you@example.com --admin-pass … \
#     --target helvishzima.local --user helvijs --pass …
#
# It installs the agent, waits for a real push to arrive, asserts the binary on
# the target matches the one the server serves, then uninstalls and asserts the
# unit is gone. It touches nothing on the target except the agent.
set -euo pipefail

SERVER="" ADMIN="" ADMIN_PASS="" TARGET="" USER_NAME="" PASS="" PORT=22 KEEP=0
while [ $# -gt 0 ]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --admin) ADMIN="$2"; shift 2 ;;
    --admin-pass) ADMIN_PASS="$2"; shift 2 ;;
    --target) TARGET="$2"; shift 2 ;;
    --user) USER_NAME="$2"; shift 2 ;;
    --pass) PASS="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --keep) KEEP=1; shift ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
done
for v in SERVER ADMIN ADMIN_PASS TARGET USER_NAME PASS; do
  if [ -z "${!v}" ]; then echo "missing --${v,,}; see the header for usage" >&2; exit 2; fi
done

JAR="$(mktemp)"; OUT="$(mktemp -d)"
trap 'rm -rf "$JAR" "$OUT"' EXIT
fail=0
ok()   { echo "ok: $1"; }
bad()  { echo "FAIL: $1"; fail=1; }

api() { # api METHOD PATH [BODY]
  local method="$1" path="$2" body="${3:-}"
  if [ -n "$body" ]; then
    curl -s -b "$JAR" -c "$JAR" -H "Origin: $SERVER" -H "Content-Type: application/json" \
      -X "$method" "$SERVER$path" -d "$body"
  else
    curl -s -b "$JAR" -c "$JAR" -H "Origin: $SERVER" -X "$method" "$SERVER$path"
  fi
}

jq_() { python3 -c "import json,sys; d=json.load(sys.stdin); print($1)"; }

remote() { sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
  -p "$PORT" "$USER_NAME@$TARGET" "$@" 2>/dev/null; }

echo "== signing in to $SERVER =="
api POST /api/auth/login "{\"email\":\"$ADMIN\",\"password\":\"$ADMIN_PASS\"}" >/dev/null
if [ "$(api GET /api/me | jq_ "d['email']" 2>/dev/null)" != "$ADMIN" ]; then
  echo "could not sign in as $ADMIN" >&2; exit 1
fi

HOST_NAME="live-ssh-test-$$"
echo "== creating host $HOST_NAME =="
HOST_ID="$(api POST /api/admin/hosts "{\"name\":\"$HOST_NAME\"}" | jq_ "d['host']['id']")"

# The scratch host goes away with the agent. With --keep both stay, or the
# agent would be left pushing a token for a host that no longer exists.
cleanup_host() { if [ "$KEEP" -eq 0 ]; then api DELETE "/api/admin/hosts/$HOST_ID" >/dev/null || true; fi; }
trap 'cleanup_host; rm -rf "$JAR" "$OUT"' EXIT

echo "== probing $TARGET =="
# The probe is not host-scoped: it answers before any host is chosen.
FPR="$(api POST /api/admin/ssh-probe \
  "{\"address\":\"$TARGET\",\"port\":$PORT}" | jq_ "d.get('fingerprint','')")"
if [ -n "$FPR" ]; then ok "probe returned a fingerprint ($FPR)"; else bad "probe returned no fingerprint"; fi

TARGET_JSON="{\"address\":\"$TARGET\",\"port\":$PORT,\"username\":\"$USER_NAME\",\"password\":\"$PASS\",\"sudo_password\":\"$PASS\",\"fingerprint\":\"$FPR\"}"

echo "== installing =="
INSTALL="$(api POST "/api/admin/hosts/$HOST_ID/ssh-install" "$TARGET_JSON")"
if echo "$INSTALL" | grep -q '"output"' && ! echo "$INSTALL" | grep -q '"code"'; then
  ok "install reported success"
else
  bad "install failed: $INSTALL"
fi

if [ "$(remote 'systemctl is-active reeve-agent')" = "active" ]; then
  ok "reeve-agent is active on $TARGET"
else
  bad "reeve-agent is not active on $TARGET"
fi

# The installed binary must be the one this server serves. Comparing checksums
# is the whole point: a version-string check once let a stale binary survive
# every reinstall while the install still reported success.
curl -s "$SERVER/dl/agent-linux-amd64" -o "$OUT/served"
SERVED_SUM="$(sha256sum "$OUT/served" | awk '{print $1}')"
INSTALLED_SUM="$(remote "echo '$PASS' | sudo -S sha256sum /usr/local/bin/reeve-agent" | awk '{print $1}')"
if [ -n "$INSTALLED_SUM" ] && [ "$SERVED_SUM" = "$INSTALLED_SUM" ]; then
  ok "installed binary matches the one the server serves"
else
  bad "installed binary differs from the server's (served ${SERVED_SUM:0:12}, installed ${INSTALLED_SUM:0:12})"
fi

echo "== waiting for the first push (up to 60s) =="
seen=""
for _ in $(seq 1 20); do
  seen="$(api GET /api/hosts | python3 -c "
import json,sys
for h in json.load(sys.stdin):
    if h['id'] == '$HOST_ID':
        print(h.get('last_seen_at',''))
")"
  if [ -n "$seen" ]; then break; fi
  sleep 3
done
if [ -n "$seen" ]; then
  ok "telemetry arrived (last_seen_at=$seen)"
else
  bad "no push arrived within 60s"
  echo "   the agent installed fine, so look at the path between them first:" >&2
  echo "   a firewall on the server (ufw allow $(echo "$SERVER" | sed -E 's|.*:([0-9]+).*|\1|')/tcp) drops pushes and shows as a timeout" >&2
  remote "journalctl -u reeve-agent -n 3 --no-pager 2>/dev/null || true" | sed 's/^/   /' >&2
fi

if [ "$KEEP" -eq 1 ]; then
  echo "== --keep given, leaving the agent installed =="
  exit "$fail"
fi

echo "== uninstalling =="
UNINSTALL="$(api POST "/api/admin/hosts/$HOST_ID/ssh-uninstall" "$TARGET_JSON")"
if echo "$UNINSTALL" | grep -q '"output"' && ! echo "$UNINSTALL" | grep -q '"code"'; then
  ok "uninstall reported success"
else
  bad "uninstall failed: $UNINSTALL"
fi

if [ "$(remote 'systemctl is-active reeve-agent || true')" = "active" ]; then
  bad "reeve-agent is still active after uninstall"
else
  ok "reeve-agent is stopped on $TARGET"
fi
if remote 'test -e /usr/local/bin/reeve-agent'; then
  bad "the agent binary is still on $TARGET"
else
  ok "the agent binary is gone from $TARGET"
fi

exit "$fail"
