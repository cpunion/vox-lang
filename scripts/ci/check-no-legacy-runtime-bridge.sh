#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

runtime_c_files="$(find src/std/runtime -maxdepth 1 -type f -name '*.c' | sort || true)"
if [[ -n "$runtime_c_files" ]]; then
  echo "[legacy-runtime] forbidden C sources found under src/std/runtime:" >&2
  echo "$runtime_c_files" | sed 's/^/  - /' >&2
  exit 1
fi

if ! ulimit -s unlimited >/dev/null 2>&1; then
  ulimit -s 65520 >/dev/null 2>&1 || true
fi

COMPILER_BIN="$(./scripts/ci/rolling-selfhost.sh print-bin | tail -n 1)"
OUT="${VOX_NO_LEGACY_RUNTIME_OUT:-target/debug/vox.no_legacy_runtime_guard}"
C_PATH="${OUT}.c"

rm -f "$OUT" "${OUT}.exe" "$C_PATH" "${OUT}.cache.key"

echo "[legacy-runtime] compiler: $COMPILER_BIN"
echo "[legacy-runtime] building guard output"
"$COMPILER_BIN" build --driver=tool "$OUT"

if [[ ! -f "$C_PATH" ]]; then
  echo "[legacy-runtime] expected generated C file not found: $C_PATH" >&2
  exit 1
fi

declare -a forbidden=(
  '^static void\* vox_impl_malloc\('
  '^static void\* vox_impl_realloc\('
  '^static void vox_impl_free\('
  '^static void vox_runtime_ctor\('
  '^static vox_vec_data\* vox_vec_data_new\('
  '^static vox_vec vox_vec_new\('
  '^static void vox_vec_grow\('
  '^static void vox_vec_push\('
  '^static void vox_vec_insert\('
  '^static void vox_vec_set\('
  '^static void vox_vec_clear\('
  '^static void vox_vec_extend\('
  '^static void vox_vec_pop\('
  '^static void vox_vec_remove\('
  '^static int32_t vox_vec_len\('
  '^static bool vox_vec_eq\('
  '^static void vox_vec_get\('
  '^static const char\* vox_vec_str_join\('
)

hits=""
for pat in "${forbidden[@]}"; do
  out="$(rg -n "$pat" "$C_PATH" || true)"
  if [[ -n "$out" ]]; then
    hits+=$'\n'"$out"
  fi
done

if [[ -n "$hits" ]]; then
  echo "[legacy-runtime] forbidden legacy bridge symbols found in generated C:" >&2
  echo "$hits" | sed '/^$/d' | sed 's/^/  - /' >&2
  exit 1
fi

echo "[legacy-runtime] ok: no legacy bridge symbols in generated C"
