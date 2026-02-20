# Verified-Refined Integer Types

## Scope

Defines predicate-based integer refinement using `@verified`.

Coverage ID: `S023`.

## Grammar

```vox
VerifiedType
  := "@verified(" PredicateRef ")" IntBaseType

PredicateRef
  := Ident
   | Ident "." Ident   // module alias + function name

IntBaseType
  := "i8" | "u8" | "i16" | "u16"
   | "i32" | "u32" | "i64" | "u64"
   | "isize" | "usize"
```

## Semantics

`@verified(check_fn) T` refines base integer type `T` with a boolean predicate function.
`T` may be any integer scalar (including `char`, treated as `u32` domain).

```vox
fn in_small(x: i32) -> bool { return x >= 0 && x <= 3; }
type Small = @verified(in_small) i32
```

Entering `Small` requires explicit cast (`as`) and predicate success.

## Predicate Function Requirements

The verifier function must satisfy:

- exactly one parameter,
- parameter type matches the verified base integer type,
- return type is `bool`,
- not `async`,
- no generic type/const parameters,
- no `effect`/`resource`/FFI attributes.

## Checks

- Runtime: casting into `@verified(...)` emits a predicate call; `false` triggers `panic("verified check failed")`.
- Assignability: verified type is assignable to its base integer type (widening).
- Base integer to verified type always requires explicit `as`.

## Const Behavior

- `const` cast into `@verified(...)` is evaluated at compile time.
- predicate returns `false` => compile-time error with `verified check failed`.

## Diagnostics

Typical errors:

- invalid predicate reference,
- predicate signature mismatch,
- base type is not an integer type,
- predicate check failure at runtime.

## Example

```vox
fn nonneg(x: i32) -> bool { return x >= 0; }
type NonNeg = @verified(nonneg) i32

fn clamp_in(x: i32) -> i32 {
  let v: NonNeg = x as NonNeg;
  return v as i32;
}
```
