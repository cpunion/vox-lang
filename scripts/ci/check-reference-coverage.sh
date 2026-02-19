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

missing_docs = sorted(matrix_ids - doc_ids)
missing_tests = sorted(matrix_ids - test_ids)
extra_tests = sorted(test_ids - matrix_ids)

if missing_docs:
    print("[reference] matrix IDs missing in language docs:", ", ".join(missing_docs), file=sys.stderr)
if missing_tests:
    print("[reference] matrix IDs missing SYNTAX markers in tests:", ", ".join(missing_tests), file=sys.stderr)
if extra_tests:
    print("[reference] SYNTAX markers not listed in matrix:", ", ".join(extra_tests), file=sys.stderr)

if missing_docs or missing_tests or extra_tests:
    sys.exit(1)

print(f"[reference] ok: {len(matrix_ids)} syntax IDs mapped across docs + tests")
PY
