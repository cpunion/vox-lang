# Attributes and FFI

## Scope

Defines currently supported built-in attributes and FFI annotation syntax.

Coverage IDs: `S701`, `S702`, `S703`, `S704`, `S705`, `S706`, `S707`.

## Grammar (Simplified)

```vox
Attr
  := "@build(" BuildExpr ")"
   | "@effect(" Ident ")"
   | "@resource(" Ident "," Ident ")"
   | "@cfg(" ("target_os" | "target_arch" | "target_ptr_bits") "," (Ident | StringLit | IntLit) ")"
   | "@ffi_import(" StringLit "," StringLit ")"
   | "@ffi_import(" StringLit "," StringLit "," StringLit ")"
   | "@ffi_export(" StringLit "," StringLit ")"
   | "@track_caller"
   | "@repr(" "C" ")"

AttributedFn
  := Attr* FnDecl

BuildExpr
  := BuildAtom
   | "!" BuildExpr
   | BuildExpr "&&" BuildExpr
   | BuildExpr "||" BuildExpr
   | "(" BuildExpr ")"

BuildAtom
  := Ident | IntLit
```

## Attribute Set

### Effect/Resource

- `@effect(Name)` declares capability metadata on function.
- `@resource(mode, Name)` declares resource access intent.

### Target Config

- `@build(expr)` is **file-scope only** and must appear at file header.
- `@cfg(target_os, value)` gates function on OS name.
- `@cfg(target_arch, value)` gates function on architecture name.
- `@cfg(target_ptr_bits, value)` gates function on pointer width (`32`/`64`).
- Multiple `@cfg(...)` on one function are AND-combined.
- Current keys: `target_os`, `target_arch`, `target_ptr_bits`.
- `@build(expr)` atoms currently match target tags:
  - OS: `linux` / `darwin` / `windows` / `wasm`
  - ARCH: `amd64` / `arm64` / `x86` / `wasm32`
  - FAMILY: `unix` (currently `linux` or `darwin`)
  - PTR bits: `32` / `64` / `ptr32` / `ptr64`
- File effective condition: all file `@build(...)` AND-combined.
- Final declaration condition: `file @build` AND declaration `@cfg`.

### FFI Import/Export

- `@ffi_import("abi", "symbol")` binds declaration to foreign symbol.
- `@ffi_import("wasm", "module", "symbol")` binds wasm import with module+symbol.
- `@ffi_export("abi", "symbol")` exports function symbol for foreign linkage.
- Same function can have multiple `@ffi_export` with different target.
- FFI import/export functions cannot be generic.
- FFI variadic functions are currently unsupported.

### FFI Boundary Guidance

- For byte/text payload across FFI, prefer `rawptr + usize` length parameters.
- FFI parameters may use limited inout form `&mut Scalar` (for example `&mut i32`) to map to C `T*` output parameters.
- `String` in FFI is kept as compatibility mapping for APIs that are naturally C-string based (for example path/command style APIs).
- Do not assume ordinary Vox text APIs implicitly require trailing `\\0`; NUL adaptation should be explicit at the compatibility layer when needed.

### Caller Tracking

- `@track_caller` enables caller-site metadata propagation.

### Representation

- `@repr(C)` on a struct guarantees C ABI-compatible field layout (order, alignment, padding).
- Currently Vox compiles to C so all structs already have C layout; the attribute serves as a semantic marker and will be enforced by future non-C backends.

## Placement Rules

Current enforced rules:

- `@cfg`, `@ffi_import`, `@ffi_export`, `@track_caller` are top-level function attributes.
- `@repr(C)` is only allowed on struct declarations.
- `@build` is only allowed at file scope (using it on function/impl/trait methods is rejected).
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
