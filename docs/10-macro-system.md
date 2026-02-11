# 宏系统（AST 宏，草案）

## 目标

1. 卫生宏：默认避免名称捕获。
2. 宏像函数：可测试、可组合、可复用。
3. 类型安全：展开后正常类型检查。

## 模型：宏是编译期函数（普通函数），返回 AST

AST 类型（示意）：

- `AstExpr` / `AstStmt` / `AstItem` / `AstBlock`

宏不需要独立的 `macro` 关键字，也不需要在函数上写 `comptime`。只要它是普通 `fn`，且返回类型是可展开的
AST 类型（例如 `AstExpr/AstStmt/AstItem/AstBlock`），编译器就会在宏展开阶段执行它。

```vox
fn add1(x: AstExpr) -> AstExpr {
  quote expr { $x + 1 }
}

fn main() {
  let v = add1!(41);
  println(v);
}
```

## quote / unquote

```vox
fn mk_stmt(x: AstExpr) -> AstStmt {
  quote stmt { let tmp = $x; }
}
```

规则（草案）：

- `$x` 插入 AST 值
- `$(expr)` 插入任意表达式计算得到的 AST
- 展开多轮迭代直到无宏调用或达到上限

## 调用与插入（草案）

宏调用提供两种形态：

1. `name!(...)`：在语法层捕获参数为 AST，并将返回的 AST 直接插回当前位置（expr/stmt/item）。
2. `compile!(ast)`：显式将一个 AST 值插回代码（用于组合宏或非 `name!(...)` 的场景）。

### 显式插入

```vox
let ast = some_macro_like_fn();
let v = compile!(ast);
```

## Stage2 现状（2026-02）

- `name!(...)` 语法在 stage2 进入 AST（独立于普通函数调用）。
- `name[T]!(...)` 也支持（与普通泛型调用语法保持一致）。
- 编译流水线已接入 `macroexpand` 阶段（`parse/load -> macroexpand -> typecheck`），展开按轮次执行直到固定点或到达轮次上限。
- 当前内置最小规则集：
  - `compile!(expr)`：仅 1 个值参数、无 type args，直接把 `expr` 插回当前位置（支持链式场景，如 `compile!(compile!(...))`）。
  - `panic!(msg)`：仅 1 个值参数，重写为 `panic(msg)`。
  - `assert!(cond)` / `assert!(cond, msg)`：重写为 `assert(cond)` / `assert_with(cond, msg)`。
- 除上述内置规则外，其他宏调用仍会在 `macroexpand` 阶段报错：`macro expansion is not supported in stage2 yet`。
- 这样做的目标是先稳定语法和流水线边界，再逐步接入真正的“宏作为编译期函数”执行器。
