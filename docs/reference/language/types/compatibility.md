# Type Compatibility and Assignability

## Scope

Defines high-level assignability behavior used by variable binding, argument passing,
and return type checking.

## Core Rule

A value of source type `S` is assignable to target type `T` only when:

- `S` and `T` are identical, or
- a defined coercion/conversion path exists and is explicitly requested (for example `as`), or
- `S` satisfies refinement constraints of `T`.

## Typical Cases

- Integer to integer: explicit cast generally required for cross-width/sign conversions.
- Integer/float cross-family: explicit cast required.
- `String` and `&str`: distinct ownership forms; not freely interchangeable.
- `&T` to `&mut T`: not assignable.
- `&mut T` to `&T`: follows borrow coercion rules where allowed by checker.
- Base integer to `@range(...)` type: requires refinement check.
- Base integer to `@verified(...)` type: requires explicit cast and predicate check.

## Generic Context

In generic functions and impls, compatibility is validated after type argument substitution
and `where` constraints evaluation.

## Diagnostics

Type checker reports incompatibility when assignment, argument binding, or return value
cannot satisfy target type requirements.

## Example

```vox
type Tiny = @range(0..=3) i32

fn f(x: i32, s: String, b: &str) -> i32 {
  let a: i64 = x as i64;
  let t: Tiny = x as Tiny;
  let n: i32 = b.len();
  return (a as i32) + t as i32 + n + s.len();
}
```
