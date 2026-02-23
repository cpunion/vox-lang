# Repository Guidelines

## Project Structure & Module Organization
- `src/vox/`: compiler core (parse, typecheck, IR, codegen, compile pipeline).
- `src/std/`: standard library modules (`sys`, `net`, `io`, `os`, `fs`, `time`, `runtime`).
- `tests/syntax/src/`: syntax acceptance matrix.
- `examples/`: runnable samples (`examples/c_demo` is CI smoke).
- `scripts/ci/`: gates (frozen builtins, layering checks, selfhost, syntax).
- `docs/internal/`, `docs/reference/`: architecture and language docs.

## Build, Test, and Development Commands
- `make fmt` / `make fmt-check`: format and formatting gate.
- `make test-active`: core gate for daily iteration.
- `make test`: full gate (syntax + rolling selfhost + examples).
- `./scripts/ci/rolling-selfhost.sh build|test|print-bin`: selfhost build/test/tool path.
- `make test-public-api`: public API stability gate.

## Coding Style & Naming Conventions
- Always run `vox fmt`; do not hand-format around it.
- Use `snake_case` for file names; tests use `*_test.vox`.
- Test function naming: `fn test_*() -> ()`.
- Non-test code should avoid `@cfg(...)` (CI-enforced); use platform partitioning patterns already used in repo.

## Constitution Rules (Must Follow)
- Keep layering strict and minimal: lower layers expose primitives; higher layers provide semantics.
- Prefer capability placement by abstraction level, not convenience of current call site.
- Keep `runtime/codegen/c_runtime` shrink-first: no new high-level surface unless bootstrap-critical.
- Keep compiler-internal capabilities inside compiler modules; do not leak them as std public APIs.
- Avoid duplicate API styles for the same behavior; keep one canonical surface and migration path.
- Any boundary change must include tests, docs, and CI gate alignment in the same PR.

### Constitution Interpretation (Examples)
- `std/sys` stays thin (syscall/FFI primitives); richer `String`/`Vec` semantics belong in higher std packages.
- `std/net` owns TCP/UDP lifecycle and protocol-facing abstractions.
- `std/io` owns stream/buffer traits and generic IO composition, not network ownership.
- `std/os` keeps OS-level semantics (process/env/path-like operations).
- `std/runtime` keeps only compiler-required minimal primitives.
- `c_runtime` is shrink-only; prefer moving behavior into std layers when feasible.
- Compiler-only helpers (for example `walk_vox_files`) belong in compiler/internal modules rather than std public API.

## Testing Guidelines
- Add/update colocated module tests in `src/**/**_test.vox`.
- Syntax/parser changes must add cases under `tests/syntax/src/`.
- For semantic/compiler changes, cover both typecheck and compile/smoke paths.
- Minimum local gate before push: `make fmt-check && make test-active`.
- Run `make test` before merging large refactors.

## Branch, PR, Review, CI, Merge Workflow
1. Sync main first: `git checkout main && git pull --ff-only`.
2. Create topic branch from main, e.g. `feat/...`, `fix/...`, `refactor/...`.
3. Implement in small, reviewable commits; keep refactor/test/doc changes separable when practical.
4. Push branch and open PR; include purpose, scope, risk, and test commands/results.
5. Address review comments with follow-up commits (do not ignore unresolved review threads).
6. Merge only when required reviews are complete and CI is green.

## GitHub CLI PR Body Rule (Encoding-Safe)
Do not pass long PR body text inline on command line. Use a Markdown file:

```bash
cat > /tmp/pr-body.md <<'MD'
## Summary
- ...

## Testing
- make fmt-check
- make test-active
MD

gh pr create \
  --base main \
  --head <your-branch> \
  --title "<type>: <short summary>" \
  --body-file /tmp/pr-body.md
```

This avoids shell quoting/encoding issues and keeps PR templates reusable.
