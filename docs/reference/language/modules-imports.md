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

## Diagnostics

- malformed import syntax is rejected by parser.
- duplicate/invalid import alias usage is diagnosed during type checking.

## Example

```vox
import "math" as m
import {a as aa, b} from "util"
pub struct P { pub v: i32 }
pub fn f(x: i32) -> i32 { return x + 1; }
fn main() -> i32 { let p: P = P { v: 1 }; return f(p.v); }
```
