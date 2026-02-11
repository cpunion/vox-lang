# Stage2 Backlog Next (P0/P1, no async/effect)

Rule: complete one item end-to-end (code + tests + docs + commit), then move to next item without leaving unresolved leftovers in the same scope.

## Items

1. [x] Generic variadic params end-to-end MVP: `xs: T...` typechecks/codegens as stable lowered form, with clear constraints and diagnostics.
2. [x] Generic type-param pack declaration usable end-to-end (remove skeleton rejections; define current semantics explicitly).
3. [x] Generic pack expansion design landing (call-site/type-site behavior and diagnostics consistency).
4. [x] Macro system strengthening: quote/unquote coverage parity for expression shapes and clearer unsupported diagnostics.
5. [x] Macro execution safety rails: deterministic expansion ordering and bounded recursion diagnostics hardening.
6. [x] Comptime evaluator parity pass: close remaining unsupported constant-expression gaps in the documented subset.
7. [x] IR semantic consistency pass: cast/compare edge behavior and verifier diagnostics alignment.
8. [ ] Diagnostics layering pass: tighter primary span coverage for typecheck/irgen/macro errors.
9. [ ] Testing framework UX pass: filtering/rerun/report fields consistency across engines.
10. [ ] Stdlib generic cleanup: remove repetitive non-generic wrappers where generic APIs already exist.
11. [ ] Package/dependency UX pass: lock mismatch diagnostics and remediation hints consistency.
12. [ ] Stage2 documentation convergence: update language/spec/tooling docs to match implemented behavior exactly.
