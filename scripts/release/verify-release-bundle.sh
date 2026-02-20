#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version> [platform]
example: $(basename "$0") v0.2.0 darwin-amd64
USAGE
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 1
fi

normalize_host_os() {
  local os="$1"
  case "$os" in
    linux) echo "linux" ;;
    darwin|macos) echo "darwin" ;;
    windows|mingw*|msys*|cygwin*) echo "windows" ;;
    *) echo "$os" ;;
  esac
}

normalize_host_arch() {
  local arch="$1"
  case "$1" in
    x64|amd64|x86_64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    x86|386|i386|i686) echo "x86" ;;
    *) echo "$arch" ;;
  esac
}

detect_host_os() {
  if [[ -n "${RUNNER_OS:-}" ]]; then
    echo "$(normalize_host_os "$(echo "$RUNNER_OS" | tr '[:upper:]' '[:lower:]')")"
    return 0
  fi
  echo "$(normalize_host_os "$(uname -s | tr '[:upper:]' '[:lower:]')")"
}

detect_host_arch() {
  if [[ -n "${RUNNER_ARCH:-}" ]]; then
    echo "$(normalize_host_arch "$(echo "$RUNNER_ARCH" | tr '[:upper:]' '[:lower:]')")"
    return 0
  fi
  echo "$(normalize_host_arch "$(uname -m | tr '[:upper:]' '[:lower:]')")"
}

VERSION="$1"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT/dist}"

HOST_PLATFORM="$(detect_host_os)-$(detect_host_arch)"

if [[ $# -eq 2 ]]; then
  PLATFORM="$2"
else
  PLATFORM="$HOST_PLATFORM"
fi

ARCHIVE="$DIST_DIR/vox-lang-${VERSION}-${PLATFORM}.tar.gz"
CHECKSUM="${ARCHIVE}.sha256"

if [[ ! -f "$ARCHIVE" ]]; then
  echo "[verify] archive not found: $ARCHIVE" >&2
  exit 1
fi
if [[ ! -f "$CHECKSUM" ]]; then
  echo "[verify] checksum file not found: $CHECKSUM" >&2
  exit 1
fi

EXPECTED="$(awk '{print $1}' "$CHECKSUM" | head -n 1)"
if [[ -z "$EXPECTED" ]]; then
  echo "[verify] empty checksum file: $CHECKSUM" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
fi

if [[ "$EXPECTED" != "$ACTUAL" ]]; then
  echo "[verify] checksum mismatch" >&2
  echo "[verify] expected: $EXPECTED" >&2
  echo "[verify] actual:   $ACTUAL" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/vox-release-verify.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

tar -xzf "$ARCHIVE" -C "$TMP_DIR"

BUNDLE_NAME="vox-lang-${VERSION}-${PLATFORM}"
BUNDLE_DIR="$TMP_DIR/$BUNDLE_NAME"
if [[ ! -d "$BUNDLE_DIR" ]]; then
  echo "[verify] bundle directory missing after extract: $BUNDLE_DIR" >&2
  exit 1
fi

EXE_SUFFIX=""
if [[ "$PLATFORM" == windows-* ]]; then
  EXE_SUFFIX=".exe"
fi

require_file() {
  local p="$1"
  if [[ ! -f "$p" ]]; then
    echo "[verify] required file missing: $p" >&2
    exit 1
  fi
}

require_file "$BUNDLE_DIR/bin/vox${EXE_SUFFIX}"
require_file "$BUNDLE_DIR/src/std/testing/testing.vox"
require_file "$BUNDLE_DIR/VERSION"
require_file "$BUNDLE_DIR/BOOTSTRAP_MODE"

BUNDLE_VERSION="$(tr -d '\r\n' < "$BUNDLE_DIR/VERSION")"
if [[ "$BUNDLE_VERSION" != "$VERSION" ]]; then
  echo "[verify] VERSION mismatch: expected=$VERSION actual=$BUNDLE_VERSION" >&2
  exit 1
fi

BOOTSTRAP_MODE="$(tr -d '\r\n' < "$BUNDLE_DIR/BOOTSTRAP_MODE")"
if [[ "$BOOTSTRAP_MODE" != "rolling" ]]; then
  echo "[verify] BOOTSTRAP_MODE must be rolling, actual=$BOOTSTRAP_MODE" >&2
  exit 1
fi

VERIFY_EXEC="${VOX_VERIFY_EXEC:-auto}"
RUN_EXEC="0"
SKIP_REASON=""
if [[ "$VERIFY_EXEC" == "1" || "$VERIFY_EXEC" == "true" ]]; then
  RUN_EXEC="1"
elif [[ "$VERIFY_EXEC" == "0" || "$VERIFY_EXEC" == "false" ]]; then
  RUN_EXEC="0"
  SKIP_REASON="VOX_VERIFY_EXEC=${VERIFY_EXEC}"
elif [[ "$PLATFORM" == "$HOST_PLATFORM" ]]; then
  RUN_EXEC="1"
else
  SKIP_REASON="non-host platform $PLATFORM"
fi

BIN_VERSION_RAW=""
if [[ "$RUN_EXEC" == "1" ]]; then
  BIN_VERSION_RAW="$($BUNDLE_DIR/bin/vox${EXE_SUFFIX} version | tr -d '\r\n')"
  EXPECTED_BIN_VERSION="vox $VERSION"
  EXPECTED_BIN_VERSION_NO_V="vox ${VERSION#v}"
  if [[ "$BIN_VERSION_RAW" != "$EXPECTED_BIN_VERSION" && "$BIN_VERSION_RAW" != "$EXPECTED_BIN_VERSION_NO_V" ]]; then
    echo "[verify] embedded version mismatch: got=$BIN_VERSION_RAW expected=$EXPECTED_BIN_VERSION or $EXPECTED_BIN_VERSION_NO_V" >&2
    exit 1
  fi
fi

echo "[verify] bundle OK: $ARCHIVE"
echo "[verify] checksum OK: $CHECKSUM"
echo "[verify] bootstrap mode: $BOOTSTRAP_MODE"
if [[ "$RUN_EXEC" == "1" ]]; then
  echo "[verify] embedded version: $BIN_VERSION_RAW"
else
  if [[ -z "$SKIP_REASON" ]]; then
    SKIP_REASON="not executable in current environment"
  fi
  echo "[verify] embedded version: skipped ($SKIP_REASON)"
fi
