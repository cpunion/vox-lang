# Generics

Current covered forms:

- generic type parameters and explicit type arguments
- const generics
- `where` trait bounds
- `where comptime` bounds
- impl-head `where comptime`
- type pack / variadic parameter surface syntax (`T...`)

Example:

```vox
fn id[T](x: T) -> T where T: Eq { return x; }
fn addn[const N: i32](x: i32) -> i32 where comptime N > 0 { return x + N; }
fn sum[T](head: T, tail: T...) -> T { return head; }
trait Tag { fn tag(x: Self) -> i32; }
impl[T] Tag for Vec[T] where comptime @size_of(T) <= 16 { fn tag(x: Vec[T]) -> i32 { return 1; } }
```
