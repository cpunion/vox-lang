#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
COMPILER_BIN="${1:-}"

if [[ -z "${COMPILER_BIN}" ]]; then
  if [[ -x "${ROOT_DIR}/target/debug/vox_rolling" ]]; then
    COMPILER_BIN="${ROOT_DIR}/target/debug/vox_rolling"
  elif [[ -x "${ROOT_DIR}/target/release/vox" ]]; then
    COMPILER_BIN="${ROOT_DIR}/target/release/vox"
  else
    echo "[wasm-smoke] compiler not found (expected target/debug/vox_rolling or target/release/vox)" >&2
    exit 1
  fi
fi

if [[ "${COMPILER_BIN}" != /* ]]; then
  COMPILER_BIN="${ROOT_DIR}/${COMPILER_BIN}"
fi
if [[ ! -x "${COMPILER_BIN}" ]]; then
  echo "[wasm-smoke] compiler not executable: ${COMPILER_BIN}" >&2
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "[wasm-smoke] node not found" >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "[wasm-smoke] curl not found" >&2
  exit 1
fi

EXAMPLE_DIR="${ROOT_DIR}/examples/wasm_call"
if [[ ! -d "${EXAMPLE_DIR}" ]]; then
  echo "[wasm-smoke] example not found: ${EXAMPLE_DIR}" >&2
  exit 1
fi

echo "[wasm-smoke] compiler: ${COMPILER_BIN}"
echo "[wasm-smoke] build wasm example"
(
  cd "${EXAMPLE_DIR}"
  "${COMPILER_BIN}" build-pkg --target=wasm32-unknown-unknown target/vox_wasm_demo.wasm
)

echo "[wasm-smoke] node call"
(
  cd "${EXAMPLE_DIR}"
  node node/run.mjs
)

echo "[wasm-smoke] web static server"
PORT="${VOX_WASM_WEB_PORT:-18080}"
SERVER_LOG="$(mktemp)"
PID_FILE="$(mktemp)"
(
  cd "${EXAMPLE_DIR}"
  VOX_WASM_WEB_PORT="${PORT}" node web/server.mjs >"${SERVER_LOG}" 2>&1 &
  echo $! >"${PID_FILE}"
)
cleanup() {
  if [[ -f "${PID_FILE}" ]]; then
    pid="$(cat "${PID_FILE}" 2>/dev/null || true)"
    if [[ -n "${pid}" ]]; then
      kill "${pid}" >/dev/null 2>&1 || true
      wait "${pid}" >/dev/null 2>&1 || true
    fi
    rm -f "${PID_FILE}"
  fi
  rm -f "${SERVER_LOG}"
}
trap cleanup EXIT

echo "[wasm-smoke] waiting for web server on :${PORT}"
ready=0
for _ in $(seq 1 20); do
  if curl -fsS "http://127.0.0.1:${PORT}/web/" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 0.25
done

if [[ "${ready}" != "1" ]]; then
  echo "[wasm-smoke] web server did not become ready" >&2
  if [[ -f "${SERVER_LOG}" ]]; then
    echo "[wasm-smoke] server log:" >&2
    cat "${SERVER_LOG}" >&2 || true
  fi
  exit 1
fi

curl -fsS "http://127.0.0.1:${PORT}/target/vox_wasm_demo.wasm" >/dev/null

echo "[wasm-smoke] ok"
