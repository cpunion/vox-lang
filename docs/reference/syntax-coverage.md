# Syntax Coverage Matrix

This matrix tracks syntax acceptance coverage.

Conventions:

- Each syntax item has an ID `Sxxx`.
- Acceptance tests live under `tests/syntax/src/`.
- Test files include `SYNTAX:Sxxx` markers.

Status:

- `Done`: covered by merged acceptance tests.
- `Partial`: partially covered; see notes.
- `Planned`: not merged yet.

## Covered (Merged)

| ID | Description | Status | Test File | Notes |
|---|---|---|---|---|
| S001 | integer and bool types | Done | `tests/syntax/src/basic_types_test.vox` |  |
| S002 | char and string literals | Done | `tests/syntax/src/basic_types_test.vox` |  |
| S003 | borrow/ref type forms | Done | `tests/syntax/src/basic_types_test.vox` | includes `&T`, `&mut T`, `&'static T`, `&'static mut T` |
| S004 | range-annotated integer type | Done | `tests/syntax/src/basic_types_test.vox` |  |
| S005 | malformed literal parse failure | Done | `tests/syntax/src/basic_types_test.vox` |  |
| S006 | struct declaration (shape/generics/where) | Done | `tests/syntax/src/adt_test.vox` |  |
| S007 | enum declaration (variants/generics/where) | Done | `tests/syntax/src/adt_test.vox` | tuple-payload and unit variants |
| S008 | struct literal and field access | Done | `tests/syntax/src/adt_test.vox` | typed path struct literal |
| S009 | enum constructor and match pattern | Done | `tests/syntax/src/adt_test.vox` | qualified variant ctor/pattern |
| S010 | malformed struct/enum declaration parse failure | Done | `tests/syntax/src/adt_test.vox` |  |
| S011 | type alias declaration | Done | `tests/syntax/src/type_alias_const_test.vox` |  |
| S012 | const declaration | Done | `tests/syntax/src/type_alias_const_test.vox` |  |
| S013 | labeled union type alias declaration | Done | `tests/syntax/src/type_alias_const_test.vox` | `type Name = A: TA | B: TB` |
| S014 | malformed type/const declaration parse failure | Done | `tests/syntax/src/type_alias_const_test.vox` |  |
| S015 | visibility markers on items | Done | `tests/syntax/src/visibility_test.vox` | `pub`, `pub(crate)`, `pub(super)` |
| S016 | visibility markers on struct fields | Done | `tests/syntax/src/visibility_test.vox` | includes `pub`, `pub(crate)`, `pub(super)` |
| S017 | malformed visibility marker parse failure | Done | `tests/syntax/src/visibility_test.vox` | invalid marker like `pub(local)` |
| S018 | triple-quoted multiline string literal rules | Done | `tests/syntax/src/basic_types_test.vox` | unindent + leading newline trim + tab-indent rejection |
| S019 | enum variant visibility marker parse failure | Done | `tests/syntax/src/visibility_test.vox` | variants cannot be prefixed by `pub`/`pub(crate)`/`pub(super)` |
| S101 | if/else statement | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S102 | if-expression | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S103 | while loop | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S104 | for-in loop | Done | `tests/syntax/src/control_flow_test.vox` | parser lowers to while+len/get |
| S105 | break/continue | Done | `tests/syntax/src/control_flow_test.vox` | includes `loop { ... }` coverage |
| S106 | match expression | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S107 | malformed control-flow parse failure | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S201 | function declaration and return | Done | `tests/syntax/src/functions_test.vox` |  |
| S202 | member call syntax | Done | `tests/syntax/src/functions_test.vox` | `i.inc()` |
| S203 | UFCS call syntax | Done | `tests/syntax/src/functions_test.vox` | `Add.add(i, 3)` |
| S204 | block expression syntax | Done | `tests/syntax/src/functions_test.vox` |  |
| S205 | malformed function declaration parse failure | Done | `tests/syntax/src/functions_test.vox` |  |
| S301 | generic params and explicit type args | Done | `tests/syntax/src/generics_test.vox` |  |
| S302 | const generics | Done | `tests/syntax/src/generics_test.vox` |  |
| S303 | where trait bounds | Done | `tests/syntax/src/generics_test.vox` |  |
| S304 | where comptime bounds | Done | `tests/syntax/src/generics_test.vox` |  |
| S305 | impl head where comptime | Done | `tests/syntax/src/generics_test.vox` |  |
| S306 | type pack and variadic params | Done | `tests/syntax/src/generics_test.vox` | includes projection form `T.N` |
| S307 | malformed generic argument parse failure | Done | `tests/syntax/src/generics_test.vox` |  |
| S401 | trait declaration | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S402 | trait default method body | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S403 | associated type in trait and impl | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S404 | supertrait syntax | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S405 | generic impl head | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S406 | negative impl syntax | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S407 | invalid negative inherent impl parse failure | Done | `tests/syntax/src/traits_impls_test.vox` |  |
| S501 | async function syntax | Done | `tests/syntax/src/async_test.vox` |  |
| S502 | postfix await syntax | Done | `tests/syntax/src/async_test.vox` |  |
| S503 | await inside if/match expressions | Done | `tests/syntax/src/async_test.vox` |  |
| S504 | async trait method syntax | Done | `tests/syntax/src/async_test.vox` |  |
| S505 | malformed await expression parse failure | Done | `tests/syntax/src/async_test.vox` |  |
| S601 | import alias form | Done | `tests/syntax/src/modules_imports_test.vox` |  |
| S602 | named import form | Done | `tests/syntax/src/modules_imports_test.vox` | `import {a as aa, b} from "util"` |
| S603 | visibility declarations in module/import flow | Done | `tests/syntax/src/modules_imports_test.vox` | includes `pub`, `pub(crate)`, `pub(super)` with imports |
| S604 | malformed import parse failure | Done | `tests/syntax/src/modules_imports_test.vox` |  |
| S701 | effect/resource attributes | Done | `tests/syntax/src/attributes_ffi_test.vox` |  |
| S702 | ffi import/export attributes | Done | `tests/syntax/src/attributes_ffi_test.vox` |  |
| S703 | track_caller attribute | Done | `tests/syntax/src/attributes_ffi_test.vox` |  |
| S704 | invalid attribute placement parse failure | Done | `tests/syntax/src/attributes_ffi_test.vox` |  |
| S801 | arithmetic operators | Done | `tests/syntax/src/operators_test.vox` |  |
| S802 | logical operators | Done | `tests/syntax/src/operators_test.vox` |  |
| S803 | bitwise/shift operators | Done | `tests/syntax/src/operators_test.vox` | includes precedence form |
| S804 | comparison/equality operators | Done | `tests/syntax/src/operators_test.vox` |  |
| S805 | cast operator | Done | `tests/syntax/src/operators_test.vox` |  |
| S806 | malformed operator expression parse failure | Done | `tests/syntax/src/operators_test.vox` |  |
| S901 | macro call sugar `name!(...)` | Done | `tests/syntax/src/macros_test.vox` |  |
| S902 | builtin macro syntax (`compile!`, `quote expr`, `$x`) | Done | `tests/syntax/src/macros_test.vox` | parser-level coverage |
| S903 | macro call with explicit generic/const args | Done | `tests/syntax/src/macros_test.vox` | `id[T]!`, `addn[N]!` |
| S904 | malformed macro call parse failure | Done | `tests/syntax/src/macros_test.vox` | missing `(` after `!` |

## Planned (Not Merged Yet)

No pending syntax IDs at the moment.
