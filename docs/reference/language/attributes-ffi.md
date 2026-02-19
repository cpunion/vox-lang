# Attributes and FFI

## Scope

Coverage IDs: `S701`, `S702`, `S703`, `S704`.

## Syntax

Effect/resource attributes:

```vox
@effect(FsRead)
@resource(read, Fs)
fn read() -> i32 { ... }
```

FFI attributes:

```vox
@ffi_import("c", "puts")
fn puts(s: String) -> i32;

@ffi_export("c", "vox_add")
fn add(a: i32, b: i32) -> i32 { ... }
```

Caller tracking:

```vox
@track_caller
fn who() -> String { ... }
```

## Semantics

- `@effect` and `@resource` annotate function-level capability and resource access metadata.
- `@ffi_import` binds a declaration to an external symbol.
- `@ffi_export` exports a function symbol for foreign linkage.
- `@track_caller` enables caller-site metadata propagation for diagnostics/logging use cases.

## Diagnostics

- `@ffi_import`, `@ffi_export`, and `@track_caller` are restricted to top-level functions.
- placing those restricted attributes in unsupported contexts (for example impl methods) is rejected.
- malformed attribute arguments are parse/typecheck errors.

## Example

```vox
@effect(FsRead)
@resource(read, Fs)
fn read() -> i32 { return 1; }

@ffi_import("c", "puts")
fn puts(s: String) -> i32;

@ffi_export("c", "vox_add")
fn add(a: i32, b: i32) -> i32 { return a + b; }

@track_caller
fn who() -> String { return "ok"; }
```
