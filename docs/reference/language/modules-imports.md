# Modules and Imports

## Scope

Coverage IDs: `S601`, `S602`, `S603`, `S604`.

## Syntax

Module alias import:

```vox
import "pkg/path" as alias
```

Named import list:

```vox
import {name1, name2 as alias2} from "pkg/path"
```

Visibility:

```vox
pub struct S { pub f: i32 }
pub fn g(x: i32) -> i32 { return x; }
```

## Semantics

- Alias imports introduce a module namespace alias.
- Named imports introduce selected names directly into current module scope.
- `pub` marks declarations visible to importing modules.

### Import Path Resolution

Current resolution model:

- Import strings are module/package paths (not filesystem-relative to current file).
- Local package modules are resolved from package source root (`src/**`).
- Dependency package modules are resolved via package manifest (`vox.toml`) dependency graph.
- In ambiguous name cases, explicit namespace prefixes are recommended (see internal module/package rules).

## Diagnostics

- malformed import syntax is rejected by parser.
- duplicate/invalid import alias usage is diagnosed during type checking.

## Example

```vox
import "math" as m
import {a as aa, b} from "util"
pub struct P { pub v: i32 }
pub fn f(x: i32) -> i32 { return m.add(aa(x), b); }
fn main() -> i32 { return f(1); }
```
