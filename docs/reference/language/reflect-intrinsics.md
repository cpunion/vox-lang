# Reflect Intrinsics

## Scope

Defines parser-level syntax for reflect/type-introspection intrinsics such as:

- size/alignment and type-id style calls (`@size_of`, `@align_of`, `@type`),
- type relation helpers (`@same_type`, `@field_name`, `@field_type`),
- predicate helpers (`@is_*` family),
- target-introspection helpers (`@target_os`, `@target_arch`, `@target_ptr_bits`).

Coverage IDs: `S906`, `S907`, `S908`.

## Grammar (Simplified)

```vox
ReflectIntrinsicCall
  := TypeReflectIntrinsicCall | TargetReflectIntrinsicCall

TypeReflectIntrinsicCall
  := "@" TypeIntrinsicName "(" TypeLikeArgList ")"

TargetReflectIntrinsicCall
  := "@" TargetIntrinsicName "(" ")"

TargetIntrinsicName := "target_os" | "target_arch" | "target_ptr_bits"
```

Examples:

```vox
@size_of(i32)
@align_of(Vec[i32])
@type(i32)
@same_type(i32, i64)
@field_name(S, 0)
@field_type(S, 1)
@is_integer(i32)
@is_vec(Vec[i32])
@is_range(R)
@target_os()
@target_arch()
@target_ptr_bits()
```

## Placement

Reflect intrinsics can appear in:

- expression positions,
- const/comptime where expressions,
- type-level contexts where a type expression accepts reflect forms (for example `@field_type(...)`).

## Diagnostics

Parser errors include malformed intrinsic call syntax (missing separators/parens).

Type/semantic arity and argument-kind checks are handled in later phases.

Target intrinsics are compile-target properties:

- `@target_os() -> String` (`linux` / `darwin` / `windows` / `wasm` / `unknown`)
- `@target_arch() -> String` (`amd64` / `arm64` / `x86` / `wasm32` / `unknown`)
- `@target_ptr_bits() -> usize` (`32` or `64`)
