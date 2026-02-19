# Vox Reference

This directory contains user-facing reference documents for Vox.

## Current Scope

Current references are aligned with the merged syntax acceptance modules:

- module-01: support harness
- module-02: basic types and literals
- module-03: control flow
- module-04: operators and casts
- module-05: functions, method call, UFCS
- module-06: generics, const generics, where/comptime, type pack syntax

## Documents

- `docs/reference/syntax.md`: syntax reference for the currently covered surface.
- `docs/reference/syntax-coverage.md`: syntax ID matrix and test mapping.

## Contribution Rule

For syntax changes, update docs and tests in the same PR:

1. update/add acceptance tests in `tests/syntax/src/*.vox` with `SYNTAX:Sxxx` markers.
2. update `docs/reference/syntax.md` and/or `docs/reference/syntax-coverage.md`.
3. resolve review comments, then merge only after CI is green.
