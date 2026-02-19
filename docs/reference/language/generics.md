# Generics

## Scope

Coverage IDs: `S301`, `S302`, `S303`, `S304`, `S305`, `S306`, `S307`.

## Syntax

- type parameters: `fn id[T](x: T) -> T`
- explicit type args: `id[i32](1)`
- const generics: `fn addn[const N: i32](x: i32) -> i32`
- trait bounds: `where T: Trait`
- comptime bounds: `where comptime <expr>`
- variadic type pack surface syntax: `tail: T...`

## Semantics

- Generic functions/types are instantiated with concrete arguments.
- `where` constraints are checked during type checking.
- malformed generic argument syntax is rejected at parse stage.

## Diagnostics

- bad bracket/argument structure in generic argument lists is a parse error.
- unsatisfied bounds are type errors.

## Example

```vox
fn id[T](x: T) -> T where T: Eq { return x; }
fn addn[const N: i32](x: i32) -> i32 where comptime N > 0 { return x + N; }
fn sum[T](head: T, tail: T...) -> T { return head; }
trait Tag { fn tag(x: Self) -> i32; }
impl[T] Tag for Vec[T] where comptime @size_of(T) <= 16 { fn tag(x: Vec[T]) -> i32 { return 1; } }
```
