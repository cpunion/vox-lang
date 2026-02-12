# Stage2 Closure (Items 1-4, 7-9)

Status: in progress.

Rule:
- Complete one item end-to-end (code + docs + tests + commit), then move to next.
- Do not reopen completed items in follow-up TODO lists.

Canonical gate for each item:
- `make fmt`
- `make test`
- item-specific tests listed below

## Checklist

1. [x] Borrow escape rules: non-static `&T` only allowed as transient parameter/local use, forbidden in return/field/container escaping positions.
   - Acceptance:
     - Add parser/typecheck tests for allowed/disallowed positions.
     - Keep `&'static T` allowed.
2. [x] String/slice transition pass: strengthen `std/string` + `std/collections` APIs for view-first usage and reduce accidental owned-copy paths.
   - Acceptance:
     - Add/adjust std tests showing view-first APIs.
     - Update docs (`07/13/14`) to exact behavior.
3. [x] Generic type-pack semantics closure: remove ambiguous skeleton behavior and lock deterministic semantics with tests.
   - Acceptance:
     - Add typecheck + compile tests that pin current supported behavior.
     - Eliminate contradictory diagnostics around pack arity/usage.
4. [x] Macro pipeline hardening: close remaining fallback ambiguity in `name!(...)` expansion path and improve deterministic diagnostics.
   - Acceptance:
     - Add macroexpand tests for fallback vs expand decisions.
     - Update macro docs to match implemented behavior exactly.
7. [ ] Diagnostics upgrade pass: tighten primary span usage and stable error code layering in remaining weak paths.
   - Acceptance:
     - Add failing tests that assert exact file/line/col + code.
8. [ ] Stage2 runtime memory convergence pass: remove known temporary runtime ownership inconsistencies and pin behavior with tests.
   - Acceptance:
     - Add runtime/codegen tests for the fixed ownership/reallocation behavior.
9. [ ] Release/rolling bootstrap operational closure: make locked-stage2 rolling path the default verified release lane.
   - Acceptance:
     - CI/release scripts + docs aligned.
     - Local dry-run command documented and passing.
