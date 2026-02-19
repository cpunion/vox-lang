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
- `pub(crate)`: visible within current package/module root scope.
- `pub(super)`: visible within parent-module scope.

## Where Visibility Appears

Current parse/type usage includes:

- top-level items (`fn`, `type`, `const`, `struct`, `enum`, `trait`)
- struct fields

## Diagnostics

Parser errors:

- invalid marker argument (for example `pub(local)`)
- malformed marker syntax

Type/check errors:

- symbol access that violates declared visibility

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
