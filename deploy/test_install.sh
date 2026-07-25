#!/usr/bin/env bash
# Unit tests for the pure helpers in install.sh (no root, no network).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"

export LV_LIB_ONLY=1
# shellcheck source=/dev/null
source "$DIR/install.sh"

fail=0
check() { if [ "$1" = "$2" ]; then echo "ok: $3"; else echo "FAIL: $3 ($1 != $2)"; fail=1; fi; }

check "$(uname_to_arch x86_64)" "amd64" "x86_64 -> amd64"
check "$(uname_to_arch aarch64)" "arm64" "aarch64 -> arm64"
check "$(uname_to_arch armv7l)" "armv7" "armv7l -> armv7"
check "$(uname_to_arch armv6l)" "armv6" "armv6l -> armv6"
check "$(uname_to_arch i686)" "386" "i686 -> 386"
check "$(uname_to_arch riscv64)" "riscv64" "riscv64 -> riscv64"
if uname_to_arch sparc64 >/dev/null 2>&1; then echo "FAIL: sparc64 should be unsupported"; fail=1; else echo "ok: sparc64 unsupported"; fi

if has_control_chars $'a\nb' >/dev/null 2>&1; then echo "ok: has_control_chars detects newline"; else echo "FAIL: has_control_chars should detect newline"; fail=1; fi
if has_control_chars "normaltoken" >/dev/null 2>&1; then echo "FAIL: has_control_chars should not flag normaltoken"; fail=1; else echo "ok: has_control_chars allows normaltoken"; fi

# Signature verification against a fixture signed with the real release key.
# Skipped where minisign is absent, which is also the path that must return 2.
FIX="$DIR/testdata/sigcheck.txt"
if command -v minisign >/dev/null 2>&1; then
  if verify_signature "$FIX" "$FIX.minisig"; then echo "ok: verify_signature accepts a valid signature"; else echo "FAIL: valid signature rejected"; fail=1; fi
  TMPD="$(mktemp -d)"
  trap 'rm -rf "$TMPD"' EXIT
  cp "$FIX.minisig" "$TMPD/x.minisig"
  printf 'tampered\n' > "$TMPD/x"
  if verify_signature "$TMPD/x" "$TMPD/x.minisig"; then echo "FAIL: tampered file accepted"; fail=1; else echo "ok: verify_signature rejects a tampered file"; fi
else
  verify_signature "$FIX" "$FIX.minisig" || rc=$?
  check "${rc:-0}" "2" "verify_signature reports 2 when minisign is missing"
fi

exit "$fail"
