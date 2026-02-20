# Attributes and FFI

## Scope

Defines currently supported built-in attributes and FFI annotation syntax.

Coverage IDs: `S701`, `S702`, `S703`, `S704`.

## Grammar (Simplified)

```vox
Attr
  := "@effect(" Ident ")"
   | "@resource(" Ident "," Ident ")"
   | "@ffi_import(" StringLit "," StringLit ")"
   | "@ffi_import(" StringLit "," StringLit "," StringLit ")"
   | "@ffi_export(" StringLit "," StringLit ")"
   | "@track_caller"

AttributedFn
  := Attr* FnDecl
```

## Attribute Set

### Effect/Resource

- `@effect(Name)` declares capability metadata on function.
- `@resource(mode, Name)` declares resource access intent.

### FFI Import/Export

- `@ffi_import("abi", "symbol")` binds declaration to foreign symbol.
- `@ffi_import("wasm", "module", "symbol")` binds wasm import with module+symbol.
- `@ffi_export("abi", "symbol")` exports function symbol for foreign linkage.
- Same function can have multiple `@ffi_export` with different target.
- FFI import/export functions cannot be generic.
- FFI variadic functions are currently unsupported.

### Caller Tracking

- `@track_caller` enables caller-site metadata propagation.

## Placement Rules

Current enforced rules:

- `@ffi_import`, `@ffi_export`, `@track_caller` are top-level function attributes.
- unsupported placement (for example impl methods) is rejected.

## Diagnostics

Parser/type errors include:

- malformed attribute argument list
- unsupported attribute target placement
- invalid ABI/symbol usage under FFI checker rules
- generic FFI function declarations
- variadic FFI function declarations (unsupported)

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
