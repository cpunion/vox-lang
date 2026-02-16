#!/usr/bin/env bash
set -euo pipefail

REPO="${VOX_INSTALL_REPO:-cpunion/vox-lang}"
MODE="download" # download | local
VERSION="${VOX_INSTALL_VERSION:-}"
PLATFORM="${VOX_INSTALL_PLATFORM:-}"
INSTALL_DIR="${VOX_INSTALL_DIR:-$HOME/.vox}"
BIN_DIR="${VOX_INSTALL_BIN_DIR:-$INSTALL_DIR/bin}"
SKIP_RC=0
FORCE=0

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { printf '[install] %s\n' "$*"; }
warn() { printf '[install] warn: %s\n' "$*" >&2; }
die() { printf '[install] error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'USAGE'
Install vox compiler.

Usage:
  bash install.sh [--local] [--version vX.Y.Z] [--platform <os-arch>]
  curl -fsSL https://raw.githubusercontent.com/cpunion/vox-lang/main/install.sh | bash
  curl -fsSL https://raw.githubusercontent.com/cpunion/vox-lang/main/install.sh | bash -s -- --version v0.2.8

Options:
  --local                 Build from local repo (rolling selfhost) then install.
  --download              Download release binary (default).
  --version <tag>         Release tag, e.g. v0.2.8 (default: latest).
  --platform <os-arch>    e.g. darwin-arm64, linux-amd64, windows-amd64.
  --repo <owner/repo>     GitHub repo (default: cpunion/vox-lang).
  --bin-dir <dir>         Install bin dir (default: ~/.vox/bin).
  --skip-rc               Do not modify shell rc files.
  --force                 Overwrite existing binary.
  -h, --help              Show help.
USAGE
}

have_cmd() { command -v "$1" >/dev/null 2>&1; }

require_cmd() {
  local c="$1"
  have_cmd "$c" || die "missing command: $c"
}

http_get() {
  local url="$1"
  local out="$2"
  if have_cmd curl; then
    curl -fsSL "$url" -o "$out"
    return 0
  fi
  if have_cmd wget; then
    wget -qO "$out" "$url"
    return 0
  fi
  die "curl or wget is required"
}

detect_platform() {
  local os arch
  os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m 2>/dev/null | tr '[:upper:]' '[:lower:]')"

  case "$os" in
    darwin) os="darwin" ;;
    linux) os="linux" ;;
    msys*|mingw*|cygwin*) os="windows" ;;
    *)
      die "unsupported os: $os (set --platform manually)"
      ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    i386|i486|i586|i686|x86) arch="x86" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      die "unsupported arch: $arch (set --platform manually)"
      ;;
  esac

  printf '%s-%s\n' "$os" "$arch"
}

platform_bin_name() {
  local p="$1"
  if [[ "$p" == windows-* ]]; then
    printf 'vox.exe\n'
  else
    printf 'vox\n'
  fi
}

normalize_version() {
  local v="$1"
  if [[ -z "$v" ]]; then
    printf '%s\n' "$v"
    return 0
  fi
  if [[ "$v" == v* ]]; then
    printf '%s\n' "$v"
    return 0
  fi
  printf 'v%s\n' "$v"
}

fetch_latest_version() {
  local api tag
  api="https://api.github.com/repos/${REPO}/releases/latest"
  local tmp
  tmp="$(mktemp)"
  http_get "$api" "$tmp"
  tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp" | head -n 1)"
  rm -f "$tmp"
  [[ -n "$tag" ]] || die "failed to detect latest release tag from $api"
  printf '%s\n' "$tag"
}

extract_release_binary_to() {
  local version="$1"
  local platform="$2"
  local out_bin="$3"
  local asset url tmp archive found

  asset="vox-lang-${version}-${platform}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${version}/${asset}"
  tmp="$(mktemp -d)"
  archive="${tmp}/${asset}"

  log "download ${url}"
  http_get "$url" "$archive"
  tar -xzf "$archive" -C "$tmp"

  found="$(find "$tmp" -type f \( -path '*/bin/vox' -o -path '*/bin/vox.exe' -o -name 'vox' -o -name 'vox.exe' \) | head -n 1 || true)"
  [[ -n "$found" ]] || die "compiler binary not found in ${asset}"

  mkdir -p "$(dirname "$out_bin")"
  cp "$found" "$out_bin"
  chmod +x "$out_bin" || true
  rm -rf "$tmp"
}

choose_rc_file() {
  local sh
  sh="${SHELL:-}"
  case "$sh" in
    */zsh)
      printf '%s\n' "$HOME/.zshrc"
      return 0
      ;;
    */bash)
      if [[ -f "$HOME/.bashrc" ]]; then
        printf '%s\n' "$HOME/.bashrc"
      else
        printf '%s\n' "$HOME/.bash_profile"
      fi
      return 0
      ;;
    */fish)
      # fish syntax differs; skip auto write.
      printf '%s\n' ""
      return 0
      ;;
  esac

  if [[ -f "$HOME/.zshrc" ]]; then
    printf '%s\n' "$HOME/.zshrc"
    return 0
  fi
  if [[ -f "$HOME/.bashrc" ]]; then
    printf '%s\n' "$HOME/.bashrc"
    return 0
  fi
  if [[ -f "$HOME/.bash_profile" ]]; then
    printf '%s\n' "$HOME/.bash_profile"
    return 0
  fi
  printf '%s\n' "$HOME/.profile"
}

ensure_path_in_rc() {
  local rc="$1"
  [[ -n "$rc" ]] || return 0
  mkdir -p "$(dirname "$rc")"
  touch "$rc"
  if grep -q '>>> vox-lang >>>' "$rc"; then
    return 0
  fi
  cat >>"$rc" <<'EOF_RC'

# >>> vox-lang >>>
export VOX_HOME="$HOME/.vox"
export PATH="$VOX_HOME/bin:$PATH"
# <<< vox-lang <<<
EOF_RC
}

install_binary() {
  local src="$1"
  local dst_name="$2"
  local dst="$BIN_DIR/$dst_name"
  mkdir -p "$BIN_DIR"
  if [[ -f "$dst" && "$FORCE" -ne 1 ]]; then
    die "target exists: $dst (use --force to overwrite)"
  fi
  cp "$src" "$dst"
  chmod +x "$dst" || true
  log "installed: $dst"
}

install_from_release() {
  local version platform bin_name tmp_bin
  platform="${PLATFORM:-$(detect_platform)}"
  version="$(normalize_version "${VERSION}")"
  if [[ -z "$version" ]]; then
    version="$(fetch_latest_version)"
  fi
  bin_name="$(platform_bin_name "$platform")"
  tmp_bin="$(mktemp)"

  extract_release_binary_to "$version" "$platform" "$tmp_bin"
  install_binary "$tmp_bin" "$bin_name"
  rm -f "$tmp_bin"
  log "release installed: ${version} (${platform})"
}

read_bootstrap_tag() {
  local repo_root="$1"
  local lock="$repo_root/scripts/release/bootstrap.lock"
  [[ -f "$lock" ]] || die "bootstrap lock not found: $lock"
  local tag
  tag="$(sed -n 's/^BOOTSTRAP_TAG="\([^"]*\)"/\1/p' "$lock" | head -n 1)"
  [[ -n "$tag" ]] || die "BOOTSTRAP_TAG missing in $lock"
  printf '%s\n' "$tag"
}

ensure_local_bootstrap() {
  local repo_root="$1"
  local platform="$2"
  local boot_noext="$repo_root/target/bootstrap/vox_prev"
  local boot_exe="$repo_root/target/bootstrap/vox_prev.exe"
  if [[ -n "${VOX_BOOTSTRAP:-}" && -f "${VOX_BOOTSTRAP}" ]]; then
    printf '%s\n' "$VOX_BOOTSTRAP"
    return 0
  fi
  if [[ -f "$boot_noext" ]]; then
    printf '%s\n' "$boot_noext"
    return 0
  fi
  if [[ -f "$boot_exe" ]]; then
    printf '%s\n' "$boot_exe"
    return 0
  fi

  mkdir -p "$repo_root/target/bootstrap"
  local tag
  tag="$(read_bootstrap_tag "$repo_root")"
  log "bootstrap not found, downloading locked bootstrap ${tag}"

  local tmp_bin
  tmp_bin="$(mktemp)"
  extract_release_binary_to "$tag" "$platform" "$tmp_bin"
  cp "$tmp_bin" "$boot_noext"
  chmod +x "$boot_noext" || true
  cp "$tmp_bin" "$boot_exe" || true
  rm -f "$tmp_bin"
  printf '%s\n' "$boot_noext"
}

install_from_local_build() {
  local repo_root platform bootstrap built bin_name
  repo_root="$SCRIPT_DIR"
  [[ -f "$repo_root/vox.toml" ]] || die "--local requires running from repo root (vox.toml not found)"
  [[ -f "$repo_root/scripts/ci/rolling-selfhost.sh" ]] || die "--local requires scripts/ci/rolling-selfhost.sh"

  platform="${PLATFORM:-$(detect_platform)}"
  bootstrap="$(ensure_local_bootstrap "$repo_root" "$platform")"
  log "using bootstrap: $bootstrap"

  (
    cd "$repo_root"
    VOX_BOOTSTRAP="$bootstrap" ./scripts/ci/rolling-selfhost.sh build
  )

  built="$repo_root/target/debug/vox_rolling"
  if [[ ! -f "$built" && -f "${built}.exe" ]]; then
    built="${built}.exe"
  fi
  [[ -f "$built" ]] || die "local build output not found: $built"

  bin_name="$(platform_bin_name "$platform")"
  install_binary "$built" "$bin_name"
  log "local build installed (${platform})"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --local)
      MODE="local"
      shift
      ;;
    --download)
      MODE="download"
      shift
      ;;
    --version)
      [[ $# -ge 2 ]] || die "missing value for --version"
      VERSION="$2"
      shift 2
      ;;
    --platform)
      [[ $# -ge 2 ]] || die "missing value for --platform"
      PLATFORM="$2"
      shift 2
      ;;
    --repo)
      [[ $# -ge 2 ]] || die "missing value for --repo"
      REPO="$2"
      shift 2
      ;;
    --bin-dir)
      [[ $# -ge 2 ]] || die "missing value for --bin-dir"
      BIN_DIR="$2"
      shift 2
      ;;
    --skip-rc)
      SKIP_RC=1
      shift
      ;;
    --force)
      FORCE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown arg: $1 (use --help)"
      ;;
  esac
done

require_cmd tar

if [[ "$MODE" == "local" ]]; then
  install_from_local_build
else
  install_from_release
fi

if [[ "$SKIP_RC" -ne 1 ]]; then
  rc_file="$(choose_rc_file)"
  if [[ -n "${rc_file}" ]]; then
    ensure_path_in_rc "$rc_file"
    log "PATH snippet ensured in: $rc_file"
  else
    warn "fish shell detected; skipped rc update. Please add ${BIN_DIR} to PATH manually."
  fi
fi

export PATH="$BIN_DIR:$PATH"
log "current shell PATH updated: $BIN_DIR"

if have_cmd vox; then
  log "vox in PATH: $(command -v vox)"
fi
if [[ -x "$BIN_DIR/vox" ]]; then
  "$BIN_DIR/vox" version || true
elif [[ -x "$BIN_DIR/vox.exe" ]]; then
  "$BIN_DIR/vox.exe" version || true
fi

cat <<EOF_DONE

Install complete.
- binary dir: ${BIN_DIR}

If current terminal still cannot find vox, run:
  source "$(choose_rc_file)"
EOF_DONE
