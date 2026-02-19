# Visibility and Access Levels

## Scope

Defines declaration visibility markers currently supported by Vox item syntax.

Coverage IDs: `S015`, `S016`, `S017`.

## Grammar (Simplified)

```vox
Vis
  := "pub"
   | "pub(crate)"
   | "pub(super)"

VisibleItem
  := Vis? ItemDecl

VisibleField
  := Vis? Ident ":" Type
```

## Visibility Levels

- private (no marker): visible only within owning module scope.
- `pub`: publicly visible.
- `pub(crate)`: visible within the current package module tree (`src/**`).
  - `tests/**` is treated as external test modules and does not get `pub(crate)` visibility from `src/**`.
- `pub(super)`: visible within the parent module and its submodules.

## Where Visibility Appears

Current parse/type usage includes:

- top-level items (`fn`, `type`, `const`, `struct`, `enum`, `trait`)
- struct fields

Notes:

- enum variants do not currently carry independent visibility markers; they follow enum-level visibility/model rules.
- `impl` blocks are not modeled with independent visibility markers.

## Diagnostics

Parser errors:

- invalid marker argument (for example `pub(local)`)
- malformed marker syntax

Type/check errors:

- symbol access that violates declared visibility
- representative messages:
  - `fn is private: <name>`
  - `const is private: <name>`
  - `struct field is private: <field>`
  - `type is private`

Import-path visibility checks use import diagnostics (see `modules-imports.md`):

- `E_IMPORT_0009`: imported name exists but is private from current module scope

## Example

```vox
pub(crate) const N: i32 = 1
pub(super) fn get() -> i32 { return N; }

pub struct S {
  pub(crate) x: i32,
  pub(super) y: i32,
  pub z: i32,
}

fn main() -> i32 {
  let s: S = S { x: 1, y: 2, z: 3 };
  return get() + s.x + s.y + s.z;
}
```
