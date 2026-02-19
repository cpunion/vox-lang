# Generics

## Scope

Defines generic parameters, const generics, `where` constraints, comptime constraints,
and current pack syntax surface.

Coverage IDs: `S301`, `S302`, `S303`, `S304`, `S305`, `S306`, `S307`.

## Grammar (Simplified)

```vox
GenericParams
  := "[" GenericParamList "]"

GenericParam
  := TypeParam
   | ConstParam

TypeParam
  := Ident

ConstParam
  := "const" Ident ":" Type

WhereClause
  := "where" WhereItem ("," WhereItem)*

WhereItem
  := Type ":" Trait
   | "comptime" Expr
```

Variadic pack surface syntax (current parser coverage):

```vox
fn sum[T](head: T, tail: T...) -> T { ... }
```

## Generic Instantiation

- Generic functions/types are instantiated with concrete type/const arguments.
- Explicit type arguments are supported (`id[i32](1)`).
- Const generic arguments are validated against declared const parameter type.

## Constraints

### Trait Bounds

`where T: Trait` requires `T` to satisfy trait implementation.

### Comptime Bounds

`where comptime <expr>` must evaluate to a compile-time truthy condition.

### Impl-Head Constraints

Constraints can appear on impl heads, including comptime predicates.

## Current Limitations

- Pack/type-variadic semantics are partially implemented; syntax surface exists.
- Advanced specialization ordering beyond current implementation is documented in internal design docs.

## Diagnostics

Parser errors:

- malformed generic parameter/argument lists
- malformed `where` syntax

Type/check errors:

- unsatisfied trait bounds
- unsatisfied comptime bounds
- const argument/type mismatches

## Example

```vox
fn id[T](x: T) -> T where T: Eq { return x; }
fn addn[const N: i32](x: i32) -> i32 where comptime N > 0 { return x + N; }
fn sum[T](head: T, tail: T...) -> T { return head; }

trait Tag { fn tag(x: Self) -> i32; }
impl[T] Tag for Vec[T] where comptime @size_of(T) <= 16 {
  fn tag(x: Vec[T]) -> i32 { return 1; }
}
```
