# vox/* Compiler Libraries

This directory hosts compiler libraries intended for reuse.

Stability tiers:

- Stable: `vox/token`, `vox/ast`, `vox/lex`, `vox/parse`, `vox/manifest`, `vox/ir`
- Experimental: `vox/types`, `vox/typecheck`, `vox/macroexpand`, `vox/irgen`, `vox/codegen`, `vox/compile`, `vox/loader`, `vox/list`
- Internal (no compatibility guarantee): `vox/internal/*` (for compiler-only shared helpers)

Policy details: `docs/internal/28-vox-libraries.md`.
