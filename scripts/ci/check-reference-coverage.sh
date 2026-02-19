#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

python - <<'PY'
import pathlib
import re
import sys

matrix_path = pathlib.Path("docs/reference/syntax-coverage.md")
lang_dir = pathlib.Path("docs/reference/language")
syntax_test_dir = pathlib.Path("tests/syntax/src")

if not matrix_path.exists():
    print("[reference] missing syntax coverage matrix:", matrix_path, file=sys.stderr)
    sys.exit(1)
if not lang_dir.exists():
    print("[reference] missing language reference dir:", lang_dir, file=sys.stderr)
    sys.exit(1)
if not syntax_test_dir.exists():
    print("[reference] missing syntax test dir:", syntax_test_dir, file=sys.stderr)
    sys.exit(1)

matrix_text = matrix_path.read_text(encoding="utf-8")
matrix_ids = set(re.findall(r"\|\s*(S\d{3})\s*\|", matrix_text))

lang_text = "\n".join(p.read_text(encoding="utf-8") for p in lang_dir.rglob("*.md"))
doc_ids = set(re.findall(r"S\d{3}", lang_text))

test_text = "\n".join(p.read_text(encoding="utf-8") for p in syntax_test_dir.rglob("*.vox"))
test_ids = set(re.findall(r"SYNTAX:(S\d{3})", test_text))

# Per-row matrix checks: valid test file path + marker presence in mapped file.
row_path_errors = []
row_marker_errors = []
for line in matrix_text.splitlines():
    if not re.match(r"^\|\s*S\d{3}\s*\|", line):
        continue
    cols = [c.strip() for c in line.strip().split("|")[1:-1]]
    if len(cols) < 4:
        continue
    sid = cols[0]
    test_col = cols[3]
    m = re.search(r"`([^`]+)`", test_col)
    if not m:
        row_path_errors.append(f"{sid}: missing backticked test file path in matrix row")
        continue
    rel = m.group(1)
    p = pathlib.Path(rel)
    if p.is_absolute():
        row_path_errors.append(f"{sid}: absolute path is not allowed: {rel}")
        continue
    if not rel.startswith("tests/syntax/src/"):
        row_path_errors.append(f"{sid}: test file must be under tests/syntax/src/: {rel}")
        continue
    if p.suffix != ".vox":
        row_path_errors.append(f"{sid}: test file must be a .vox file: {rel}")
        continue
    if not p.exists():
        row_path_errors.append(f"{sid}: mapped test file does not exist: {rel}")
        continue
    file_text = p.read_text(encoding="utf-8")
    if f"SYNTAX:{sid}" not in file_text:
        row_marker_errors.append(f"{sid}: mapped test file missing marker SYNTAX:{sid} ({rel})")

missing_docs = sorted(matrix_ids - doc_ids)
missing_tests = sorted(matrix_ids - test_ids)
extra_tests = sorted(test_ids - matrix_ids)

if missing_docs:
    print("[reference] matrix IDs missing in language docs:", ", ".join(missing_docs), file=sys.stderr)
if missing_tests:
    print("[reference] matrix IDs missing SYNTAX markers in tests:", ", ".join(missing_tests), file=sys.stderr)
if extra_tests:
    print("[reference] SYNTAX markers not listed in matrix:", ", ".join(extra_tests), file=sys.stderr)
if row_path_errors:
    print("[reference] invalid Test File mapping rows:", file=sys.stderr)
    for e in row_path_errors:
        print("  -", e, file=sys.stderr)
if row_marker_errors:
    print("[reference] per-row SYNTAX marker mapping errors:", file=sys.stderr)
    for e in row_marker_errors:
        print("  -", e, file=sys.stderr)

if missing_docs or missing_tests or extra_tests or row_path_errors or row_marker_errors:
    sys.exit(1)

print(f"[reference] ok: {len(matrix_ids)} syntax IDs mapped across docs + tests")
PY
