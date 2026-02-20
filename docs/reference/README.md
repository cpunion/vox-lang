# Vox Reference

This directory contains user-facing reference documents for Vox.

## Entry Points

- `docs/reference/language/README.md`: language reference tree.
- `docs/reference/syntax.md`: syntax index (redirect to topic pages).
- `docs/reference/syntax-coverage.md`: syntax ID matrix and test mapping.
- `docs/reference/style-guide.md`: project execution style guide (design/API/testing/PR workflow).

## Reference Tree

- Type system:
  - `docs/reference/language/types.md`
  - `docs/reference/language/types/*.md`
- Control flow:
  - `docs/reference/language/control-flow/README.md`
  - `docs/reference/language/control-flow/*.md`
- Expressions/calls/modules:
  - `docs/reference/language/operators.md`
  - `docs/reference/language/error-handling.md`
  - `docs/reference/language/reflect-intrinsics.md`
  - `docs/reference/language/functions.md`
  - `docs/reference/language/macros.md`
  - `docs/reference/language/modules-imports.md`
  - `docs/reference/language/visibility.md`
- Generics/traits/async/attributes:
  - `docs/reference/language/generics.md`
  - `docs/reference/language/traits-impls.md`
  - `docs/reference/language/async-await.md`
  - `docs/reference/language/attributes-ffi.md`
- Constants:
  - `docs/reference/language/constants.md`

## Contribution Rule

For syntax changes, update docs and tests in the same PR:

1. update/add acceptance tests in `tests/syntax/src/*.vox` with `SYNTAX:Sxxx` markers.
2. update corresponding topic docs under `docs/reference/language/**` and `docs/reference/syntax-coverage.md`.
3. run `make test-reference test-syntax` locally.
4. resolve review comments, then merge only after CI is green.
