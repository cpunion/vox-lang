# References and Borrows

## Scope

Defines borrow/reference type forms currently available in Vox syntax.

Coverage ID: `S003`.

## Grammar

```vox
RefType
  := "&" Type
   | "&mut" Type
   | "&'static" Type
   | "&'static mut" Type
```

## Forms

- `&T`: shared immutable borrow.
- `&mut T`: exclusive mutable borrow.
- `&'static T`: shared borrow with static lifetime marker.
- `&'static mut T`: mutable static borrow.

## Lifetime Surface

Current syntax supports only explicit `'static` lifetime marker in type positions.
User-defined named lifetime parameters are not part of public language syntax.

## Borrow Semantics

- Reading through `&T` is allowed.
- Writing through `&mut T` is allowed.
- Borrowing and dereference behavior is validated by type checking and lowering rules.

## Type Errors

Type checking rejects:

- writes through immutable borrows,
- incompatible borrow type assignments,
- invalid dereference usage.

## Example

```vox
fn bump(x: &mut i32) -> () {
  *x = *x + 1;
}

fn view(x: &i32, s: &'static str) -> i32 {
  return *x + s.len();
}
```
