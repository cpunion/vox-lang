# Traits and Impls

## Scope

Coverage IDs: `S401`, `S402`, `S403`, `S404`, `S405`, `S406`, `S407`.

## Syntax

Trait declaration:

```vox
trait Name[: SuperTrait] {
  type AssocType;
  fn method(x: Self) -> Ret;
  fn defaulted(x: Self) -> Ret { ... }
}
```

Impl forms:

```vox
impl Trait for Type { ... }
impl[T] Trait for Type[T] { ... }
impl !Trait for Type {}
```

## Semantics

- Trait methods define required behavior contracts for implementers.
- Default method bodies can be provided in trait definitions.
- Associated types are bound in concrete impl blocks.
- Negative impls declare non-implementation for auto-trait-like constraints.

## Diagnostics

- Invalid negative inherent impl forms are rejected (for example `impl !Type {}`).
- Trait/impl signature mismatches are reported by type checking.

## Example

```vox
trait Base { fn base(x: Self) -> i32; }
trait Iter: Base {
  type Item;
  fn next(x: Self) -> Self.Item;
  fn ready(x: Self) -> bool { return true; }
}
struct I { v: i32 }
impl Base for I { fn base(x: I) -> i32 { return x.v; } }
impl Iter for I {
  type Item = i32;
  fn next(x: I) -> i32 { return x.v; }
}
trait Show { fn show(x: Self) -> String; }
impl[T] Show for Vec[T] {
  fn show(x: Vec[T]) -> String { return x.len().to_string(); }
}
impl !Send for I {}
```
