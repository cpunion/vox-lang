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
- `macroexpand` 默认轮次上限为 `512`（`ExpandConfig.max_rounds`），避免“宏数量较多但可收敛”场景被过早截断；仍可通过配置收紧。
- 当前内置最小规则集：
  - `compile!(expr)`：仅 1 个值参数、无 type args，直接把 `expr` 插回当前位置（支持链式场景，如 `compile!(compile!(...))`）。
  - `panic!(msg)`：仅 1 个值参数，重写为 `panic(msg)`。
  - `compile_error!(msg)`：仅 1 个值参数，重写为 `@compile_error(msg)`。
  - `assert!(cond)` / `assert!(cond, msg)`：重写为 `assert(cond)` / `assert_with(cond, msg)`。
  - 比较断言宏（均要求 2 个值参数、无 type args）：
    - `assert_eq!(a, b)` -> `assert_eq(a, b)`
    - `assert_ne!(a, b)` -> `assert_ne(a, b)`
    - `assert_lt!(a, b)` -> `assert_lt(a, b)`
    - `assert_le!(a, b)` -> `assert_le(a, b)`
    - `assert_gt!(a, b)` -> `assert_gt(a, b)`
    - `assert_ge!(a, b)` -> `assert_ge(a, b)`
- 非内置宏调用采用“内联优先 + 调用糖回退”的过渡语义：
  - 先尝试模板内联：
    - 本模块：`name!(...)` 对应到本模块函数 `fn name(...)`，且函数体满足“末尾 `return <expr>;` 模板”：
      - 单语句函数体（`return <expr>;`）直接以内联表达式模板处理。
      - 多语句函数体会被视为模板块 `{ stmt*; tail }`（其中 `tail` 为最后一条 `return` 的表达式）再内联。
      - 若前置语句中存在 `return`（含嵌套 `if/while` 分支），为避免控制流语义歧义，回退到调用糖。
    - `return` 表达式现已支持块表达式模板（如 `return { let y = x + 1; y };`）以及 `match` / `try {}` 模板内联。
    - 泛型模板支持显式实参替换：`name[T, 3]!(...)` 会在内联时同步替换模板中的类型参数与常量参数（缺少显式泛型实参时回退）。
    - 跨模块（`pkg.name!(...)`）：仅在模板对跨模块内联安全时启用（当前要求模板表达式只能依赖形参与其派生表达式），否则回退。
  - 其余情况回退为调用糖：`name!(...)` / `name[T]!(...)` -> `name(...)` / `name[T](...)`，再进入常规 typecheck。
- 这样做的目标是先稳定语法和流水线边界，并让“宏即函数”的调用形态可直接使用；后续再把返回 AST 的真正宏执行器接入同一阶段。
