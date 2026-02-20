# Generics

## Scope

Defines generic parameters, const generics, `where` constraints, comptime constraints,
and type pack syntax/materialization.

Coverage IDs: `S301`, `S302`, `S303`, `S304`, `S305`, `S306`, `S307`, `S308`.

## Grammar (Simplified)

```vox
GenericParams
  := "[" GenericParamList ","? "]"

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

Pack projection syntax:

```vox
fn pick_first[T...](a: T.0, _b: T.1) -> T.0 { return a; }
```

## Generic Instantiation

- Generic functions/types are instantiated with concrete type/const arguments.
- Explicit type arguments are supported (`id[i32](1)`).
- Generic bracket argument lists accept an optional trailing comma (`id[i32, 1,](x)`).
- Const generic arguments are validated against declared const parameter type.
- Trailing explicit type arguments can bind a trailing type pack (including heterogeneous packs).

## Type Packs

- Declaration form: `T...` in generic parameter list.
- Variadic parameter form: `xs: T...`.
- Projection form: `T.N` (for example `T.0`, `T.1`) in type positions.
- Pack substitution/materialization is applied consistently across type checking, const evaluation, and IR generation for supported generic call paths.

## Constraints

### Trait Bounds

`where T: Trait` requires `T` to satisfy trait implementation.

### Comptime Bounds

`where comptime <expr>` must evaluate to a compile-time truthy condition.

### Impl-Head Constraints

Constraints can appear on impl heads, including comptime predicates.
Specialization ranking treats stronger comptime inequality bounds as more specific
when they target the same lhs/rhs basis (for example `<= 8` is stronger than `<= 16`).

## Current Limits

- Pack materialization enforces an arity limit; exceeding it is a type error.
- Specialization ordering follows current checker ranking rules and rejects incomparable overlaps.

## Diagnostics

Parser errors:

- malformed generic parameter/argument lists
- malformed `where` syntax

Type/check errors:

- unsatisfied trait bounds
- unsatisfied comptime bounds
- const argument/type mismatches
- `type pack arity exceeds materialization limit`

## Example

```vox
fn id[T](x: T) -> T where T: Eq { return x; }
fn addn[const N: i32](x: i32) -> i32 where comptime N > 0 { return x + N; }
fn sum[T](head: T, tail: T...) -> T { return head; }
fn pick_first[T...](a: T.0, _b: T.1) -> T.0 { return a; }

trait Tag { fn tag(x: Self) -> i32; }
impl[T] Tag for Vec[T] where comptime @size_of(T) <= 16 {
  fn tag(x: Vec[T]) -> i32 { return 1; }
}
```
