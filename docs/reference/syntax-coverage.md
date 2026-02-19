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
| S003 | borrow/ref type forms | Done | `tests/syntax/src/basic_types_test.vox` | includes `&T`, `&'static str` |
| S004 | range-annotated integer type | Done | `tests/syntax/src/basic_types_test.vox` |  |
| S005 | malformed literal parse failure | Done | `tests/syntax/src/basic_types_test.vox` |  |
| S101 | if/else statement | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S102 | if-expression | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S103 | while loop | Done | `tests/syntax/src/control_flow_test.vox` |  |
| S105 | break/continue | Partial | `tests/syntax/src/control_flow_test.vox` | `loop` keyword coverage pending |
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
| S306 | type pack and variadic params | Done | `tests/syntax/src/generics_test.vox` |  |
| S307 | malformed generic argument parse failure | Done | `tests/syntax/src/generics_test.vox` |  |
| S801 | arithmetic operators | Done | `tests/syntax/src/operators_test.vox` |  |
| S802 | logical operators | Done | `tests/syntax/src/operators_test.vox` |  |
| S803 | bitwise/shift operators | Done | `tests/syntax/src/operators_test.vox` | includes precedence form |
| S804 | comparison/equality operators | Done | `tests/syntax/src/operators_test.vox` |  |
| S805 | cast operator | Done | `tests/syntax/src/operators_test.vox` |  |
| S806 | malformed operator expression parse failure | Done | `tests/syntax/src/operators_test.vox` |  |

## Planned (Not Merged Yet)

| ID | Description | Status |
|---|---|---|
| S104 | for-in loop | Planned |
| S401-S407 | traits and impls syntax coverage module | Planned |
| S501-S505 | async syntax coverage module | Planned |
| S601-S604 | modules/imports syntax coverage module | Planned |
| S701-S704 | attributes/ffi syntax coverage module | Planned |
