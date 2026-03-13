#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $(basename "$0") <bootstrap-bin-or-name>" >&2
  exit 1
fi

bootstrap_base="$(basename "$1")"

truthy_01() {
  local raw="${1:-}"
  local s=""
  s="$(echo "$raw" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$s" in
    1|true|yes|on) echo "1" ;;
    *) echo "0" ;;
  esac
}

legacy_runtime_c="${VOX_LEGACY_C_RUNTIME:-}"
if [[ -n "$legacy_runtime_c" ]]; then
  truthy_01 "$legacy_runtime_c"
  exit 0
fi

# Locked bootstrap binaries named vox_prev still require runtime C bridge.
if [[ "$bootstrap_base" == "vox_prev" || "$bootstrap_base" == "vox_prev.exe" ]]; then
  echo "1"
else
  echo "0"
fi
