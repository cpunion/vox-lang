# Traits and Impls

Current covered forms:

- trait declaration
- trait default method body
- associated type in trait and impl
- supertrait syntax (`trait A: B`)
- generic impl head (`impl[T] ...`)
- negative impl syntax (`impl !Trait for Type {}`)

Example:

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
