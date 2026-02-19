# Vox Reference

This directory contains user-facing reference documents for Vox.

## Current Scope

Current references are aligned with merged syntax acceptance modules, and are being
normalized into reference-style pages (grammar, semantics, diagnostics, examples):

- module-01: support harness
- module-02: basic types and literals
- module-03: control flow
- module-04: operators and casts
- module-05: functions, method call, UFCS
- module-06: generics, const generics, where/comptime, type pack syntax
- module-07: traits and impls syntax
- module-08: struct/enum declarations and ADT usage syntax

## Documents

- `docs/reference/language/README.md`: language reference tree (entrypoint).
- `docs/reference/syntax.md`: compatibility index redirecting to language tree docs.
- `docs/reference/syntax-coverage.md`: syntax ID matrix and test mapping.

Type system details are split into dedicated pages under:

- `docs/reference/language/types/`

## Contribution Rule

For syntax changes, update docs and tests in the same PR:

1. update/add acceptance tests in `tests/syntax/src/*.vox` with `SYNTAX:Sxxx` markers.
2. update corresponding topic docs under `docs/reference/language/**` and `docs/reference/syntax-coverage.md`.
3. resolve review comments, then merge only after CI is green.
