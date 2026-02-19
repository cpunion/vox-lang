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

- `pub` marks declarations as exportable outside current module/package boundary.
- non-`pub` items are module-internal.

## Path Resolution

Current model:

- import string is a module/package path identifier, not a relative filesystem expression.
- local package modules are resolved from package source roots (for example `src/**`).
- dependency modules are resolved through `vox.toml` dependency graph.

## Diagnostics

Parser errors:

- malformed `import` syntax
- malformed named import list

Type/checker errors:

- unresolved import module or symbol
- duplicate imported names in same scope
- invalid visibility usage

## Example

```vox
import "math" as m
import {a as aa, b} from "util"

pub struct P { pub v: i32 }
pub fn f(x: i32) -> i32 { return m.add(aa(x), b); }
fn main() -> i32 { return f(1); }
```
