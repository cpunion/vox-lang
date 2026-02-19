# Type System

## Scope

This section is the normative reference for Vox types.

Coverage IDs: `S001`-`S011`, `S013`-`S014`, `S018`-`S020`.

## Type Family Map

- Primitive and scalar types: `docs/reference/language/types/primitives.md`
- Type aliases and labeled union aliases: `docs/reference/language/types/aliases.md`
- String and text types: `docs/reference/language/types/text.md`
- Borrow/reference types: `docs/reference/language/types/references.md`
- Struct types: `docs/reference/language/types/structs.md`
- Enum types: `docs/reference/language/types/enums.md`
- Range-refined integer types: `docs/reference/language/types/ranges.md`
- Type conversion and cast rules: `docs/reference/language/types/conversions.md`
- Type compatibility and assignability: `docs/reference/language/types/compatibility.md`
- Literal forms and typing: `docs/reference/language/types/literals.md`

## Summary

Vox currently supports:

- Integer, float, bool, and char primitive types.
- `String` and borrowed `str` views.
- Borrow forms `&T`, `&mut T`, `&'static T`, `&'static mut T`.
- Integer range refinement through `@range(lo..=hi) BaseInt`.
- Explicit casts via `as`.

For complete grammar, typing behavior, and diagnostics, use the topic pages above.
