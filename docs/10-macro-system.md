# 宏系统（AST 宏）

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

规则：

- `$x` 插入 AST 值
- `$(expr)` 插入任意表达式计算得到的 AST
- 展开多轮迭代直到无宏调用或达到上限

MVP（当前已实现）：

- 使用 `quote!(expr)` / `unquote!(expr)` 的函数式形式，覆盖表达式位点。
- 已支持表面语法糖：
  - `quote expr { ... }` -> `quote!(...)`
  - `$x` -> `unquote!(x)`
  - 因此 `quote expr { $x + 1 }` 可直接工作，并与函数式形式等价。

## 调用与插入

宏调用提供两种形态：

1. `name!(...)`：在语法层捕获参数为 AST，并将返回的 AST 直接插回当前位置（expr/stmt/item）。
2. `compile!(ast)`：显式将一个 AST 值插回代码（用于组合宏或非 `name!(...)` 的场景）。

### 显式插入

```vox
fn make_add1(x: AstExpr) -> AstExpr { return x + 1; }
fn main() -> i32 { return compile!(make_add1(41)); }
```

## 当前现状（2026-02）

- `name!(...)` 语法在 进入 AST（独立于普通函数调用）。
- `name[T]!(...)` 也支持（与普通泛型调用语法保持一致）。
- 编译流水线已接入 `macroexpand` 阶段（`parse/load -> macroexpand -> typecheck`），展开按轮次执行直到固定点或到达轮次上限。
- `macroexpand` 默认轮次上限为 `512`（`ExpandConfig.max_rounds`），避免“宏数量较多但可收敛”场景被过早截断；仍可通过配置收紧。
  - 达到上限时的诊断会包含 `pending macro calls` 与 `next: <module>::<macro>!` 摘要，便于定位未收敛点。
- 当前内置最小规则集：
  - `compile!(expr)`：仅 1 个值参数、无 type args，直接把 `expr` 插回当前位置（支持链式场景，如 `compile!(compile!(...))`）。
    - 当 `expr` 是普通调用（`f(...)` / `pkg.f(...)`）且 `f` 返回 `Ast*` 时，会先转为宏调用再展开，支持 `compile!(make_ast(...))` 形态。
    - 上述 `f(...)` 也支持命名导入函数（`import { f as g } from "dep"; compile!(g(...))`）。
    - 支持“宏调用宏”级联展开：`compile!(m2(...))` 若 `m2` 模板中继续产生 `m1!(...)`，会在后续轮次继续展开直到收敛。
- `quote!(expr)`：仅 1 个值参数、无 type args，表达式级 quote MVP（当前直接产出内联表达式节点）。
- `unquote!(expr)`：仅 1 个值参数、无 type args，表达式级 unquote MVP（当前直接产出内联表达式节点）。
  - 覆盖形态已验证包含普通二元表达式、`if` 表达式与 `match` 表达式组合场景。
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
    - 泛型模板支持显式实参替换：`name[T, 3]!(...)` 会在内联时同步替换模板中的类型参数与常量参数。
      - 类型参数缺失时回退到调用糖（当前宏展开阶段不做类型参数推导）。
      - 常量参数允许缺省；若声明了默认值会在内联时补齐并参与替换。
    - 跨模块（`pkg.name!(...)`）：仅在模板对跨模块内联安全时启用（当前要求模板表达式只能依赖形参与其派生表达式），否则回退。
  - 其余情况回退为调用糖：`name!(...)` / `name[T]!(...)` -> `name(...)` / `name[T](...)`，再进入常规 typecheck。
  - 例外：若已定位到目标模块但找不到被调用函数（missing callee），macroexpand 直接报错，不再回退到调用糖。
  - 当发生“内联跳过并回退”时，编译失败场景会在错误文本追加 `[macroexpand] ...` 注记，包含具体跳过原因（如“generic arg count mismatch”）。
  - 展开阶段会额外记录稳定决策注记：`decision=inline-template` 或 `decision=fallback-call`，用于区分“成功内联”与“调用糖回退”。
  - 对“模板不支持 / 跨模块作用域不安全”场景，注记会带上模板根表达式类型（如 `root: Call`），便于快速定位。
- 宏执行 v1（无 `macro` 关键字）：
  - 返回类型为 `AstExpr/AstStmt/AstItem/AstBlock` 的普通 `fn` 会被当作“宏执行函数”参与 `name!(...)` 展开流程。
  - 当前 `name!(...)` 调用位点是表达式位置，因此已稳定支持 `AstExpr` 与 `AstBlock` 形态的函数式宏模板执行。
  - 当宏执行函数返回 `AstStmt/AstItem` 且在表达式位点被调用时，macroexpand 会直接报错：`macro call requires AstExpr or AstBlock return type`（避免回退路径的非确定行为）。
  - 当 `name!(...)` 或 `compile!(...)` 直接位于语句位点（`ExprStmt`）时，`AstStmt` 返回类型也可用（例如 `mk_stmt!(...)` 或 `compile!(mk_stmt(...))` 作为独立语句）。
  - `name!(...)` 支持模块别名与命名导入两种跨模块调用方式（例如 `dep.add1!(...)` / `import { add1 as inc } ...; inc!(...)`）。
  - 模板中可直接使用 `const` 泛型参数标识符（例如 `x + N`）；展开时会按调用实参替换为字面量。
  - 展开收敛后，这类宏执行函数会从 world 中剔除，不进入后续运行时 typecheck/codegen（避免把 `Ast*` 当作运行时类型）。
- 这样做的目标是先稳定语法和流水线边界，并让“宏即函数”的调用形态可直接使用；后续再把返回 AST 的真正宏执行器接入同一阶段。
