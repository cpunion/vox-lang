# Modules and Imports

## Scope

Defines module visibility, import forms, and package-path resolution behavior.

Coverage IDs: `S601`, `S602`, `S603`, `S604`.

## Grammar (Simplified)

```vox
ImportDecl
  := "import" StringLit "as" Ident
   | "import" "{" ImportItemList "}" "from" StringLit

ImportItem
  := Ident
   | Ident "as" Ident

VisDecl
  := "pub" ItemDecl
   | "pub(crate)" ItemDecl
   | "pub(super)" ItemDecl
```

## Import Forms

### Alias Import

```vox
import "pkg/path" as alias
```

Introduces a namespace alias into current module scope.

### Named Import

```vox
import {name1, name2 as alias2} from "pkg/path"
```

Brings selected exported names directly into current module scope.

## Visibility

- `pub` marks declarations as publicly visible.
- `pub(crate)` marks declarations visible inside current package.
- `pub(super)` marks declarations visible in parent module scope.
- declarations without visibility marker are private to current module.

See `docs/reference/language/visibility.md` for the full visibility model.

## Path Resolution

Current model:

- import string is a module/package path identifier, not a relative filesystem expression.
- local package modules are resolved from package source roots (for example `src/**`).
- dependency modules are resolved through `vox.toml` dependency graph.
- all `.vox` files under the same source directory belong to the same module path; file name itself is not the module name.

## Diagnostics

Parser errors:

- malformed `import` syntax
- malformed named import list

Type/checker errors:

- unresolved import module or symbol
- duplicate imported names in same scope
- invalid visibility usage

Current import diagnostics (stable codes):

- `E_IMPORT_0002`: unknown module import
- `E_IMPORT_0004`: duplicate import alias in same file
- `E_IMPORT_0005`: duplicate local name in named-import list
- `E_IMPORT_0006`: named-import local name conflicts with module alias/local declaration
- `E_IMPORT_0007`: unknown imported name in target module
- `E_IMPORT_0009`: imported name exists but is private from current module scope

## Example

```vox
import "math" as m
import {a as aa, b} from "util"

pub struct P { pub v: i32 }
pub fn f(x: i32) -> i32 { return m.add(aa(x), b); }
pub(crate) fn g(x: i32) -> i32 { return f(x); }
fn main() -> i32 { return g(1); }
```
