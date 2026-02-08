# Stage1 编译器（Vox 实现，草案）

本章描述 `compiler/stage1` 的工程结构与近期开发顺序。Stage1 的目标是在 Vox 语言子集内实现 Vox 编译器，并最终替代 Stage0（Go）。

## 目录约定

- `compiler/stage1/src/main.vox`：编译器入口（暂为占位）。
- `compiler/stage1/src/std/**`：Stage1 的标准库源码（由 Stage0 注入，用于 Stage0/Stage1 共用最小工具）。
- `compiler/stage1/src/**/**_test.vox`：Stage1 自身的单元测试，与实现代码同目录同包（由 Stage0 的 `vox test` 运行）。
- `compiler/stage1/src/compile/**`：Stage1 的最小“串联管线”（parse -> typecheck -> irgen -> codegen）入口，便于在无 IO 的前提下做端到端测试。

## 近期顺序（可迭代）

1. 词法（lexer）：把 `String` 解析为 token 流，包含位置（byte offset）。
2. 语法（parser）：从 token 流构建 AST（使用 arena/index 建模递归结构）。
3. 类型检查：覆盖 Stage0 子集。
4. IR v0：对齐 `docs/19-ir-spec.md`，跑通从 AST 到 IR。
5. 后端（C）：先复用 Stage0 的 C 后端策略（tagged union、by-value struct），产出单文件 C 源码，后续再接上编译/链接为可执行文件。

说明：

- Stage1 lexer 的 `Token` 不直接携带 `lexeme`，而是携带 `[start,end)` 的 byte offset。解析器需要通过 `source.slice(start, end)` 拉取 token 文本。
- Stage0 已提供 `String.slice(start, end) -> String` 作为过渡能力；其内存行为在 Stage0/C 后端下会产生分配（临时泄漏可接受，后续再用切片/真实字符串模型替换）。

当前进度（实现状态以代码为准）：

- lexer：已覆盖常用关键字/标点、字符串/整数、注释与错误定位（byte offset）。
- parser：已支持 `import "pkg" as alias`、`pub struct`/`enum`/`fn`，以及语句 `let`/赋值/`if`/`while`/`break`/`continue`/`return`/表达式语句；表达式包含 member/call、struct literal、`match { pat => expr }` 与常见二元/一元运算（precedence climbing）。类型名已支持 `path`（`a.b.C`）与方括号泛型（`Vec[i32]`）。
- typecheck：已覆盖 Stage0 子集的主要路径（函数调用、member 调用、struct literal、enum ctor、match、Vec/String 最小内建），并支持泛型函数签名与泛型调用（可显式 `f[T](...)`，也可从参数/返回期望推导）。
- IR v0：`compiler/stage1/src/ir/**` 已对齐 `docs/19-ir-spec.md`（TyPool、Value/Instr/Term、Program 结构与 formatter）。
- IRGen：`compiler/stage1/src/irgen/**` 已跑通 end-to-end（从 AST 生成 IR），并对泛型调用做单态化（worklist 生成可达的实例函数）。
- codegen（C）：`compiler/stage1/src/codegen/**` 已支持 IR v0 -> 单文件 C 源码；并通过 `compiler/stage1/src/compile/**` 提供最小串联管线（用于端到端测试）。
- loader（in-memory）：`compiler/stage1/src/loader/**` 已支持按 `src/`/`tests/` 目录规则把多文件合并为模块（`src/*.vox -> main`，`src/<dir>/** -> <dir>`，`tests/** -> tests/...`），用于多模块端到端编译测试。
- stage1 CLI（最小）：`compiler/stage1/src/main.vox` 提供 `emit-c/build/build-pkg`，使用 `std/fs` 与 `std/process` 完成读写与调用系统 `cc`（用于自举前的工具链验证）。CLI 会自动注入 stage1 自带的 `src/std/**` 作为被编译包的本地 `src/std/**`；`build-pkg` 还会读取当前目录 `vox.toml` 的 path 依赖并加载其 `src/**`（包含传递依赖）。另外 `emit-c/build/build-pkg` 支持 `--driver=user|tool`：`tool` 模式下生成的二进制不打印返回值，并把 `main() -> i32` 作为进程退出码返回（用于自举工具）。
- stage1 CLI（测试）：`compiler/stage1/src/main.vox` 还提供 `test-pkg`，发现并运行 `src/**/*_test.vox` 与 `tests/**/*.vox` 中的 `test_*`（行为对齐 stage0 的 `vox test`：单一测试二进制 + 每个测试单独进程运行）。

## 与自举相关的实现细节（当前实现）

### 1) 字符串转义与生成 C

- Stage1 lexer 仅负责识别 token 边界；parser 会对字符串字面量做最小的反转义（至少支持 `\\ \" \n \r \t`）。
- codegen 输出到 C 字符串字面量时，会对内容做 C 级别转义（例如把换行字符输出为 `\\n`），避免生成的 `.c` 文件被“字面量换行”破坏。

### 2) enum 相等（Stage0/Stage1 v0 约束）

为保持范围可控，Stage0/Stage1 对 `==/!=` 的支持与 lowering 有明确限制（详见 `docs/14-syntax-details.md`）：

- `bool/i32/i64/String`：正常相等比较
- `enum`：仅允许与 unit variant（无 payload）比较；该比较会降低为 `EnumTag(x) == tag(Variant)`

### 3) TyPool 在 lowering 期间的“可变性”

Stage1 的类型池（`TyPool`）在 typecheck 阶段会 intern 已知类型，但 lowering 期间仍可能因为泛型实例化/容器类型构造等引入新类型。

当前实现策略：

- IRGen 生成函数体时允许对 `Ctx.pool` 追加 intern 的类型
- `ir.Program.pool` 在 IRGen 完成后与最终的 `Ctx.pool` 同步，保证 codegen 阶段访问的类型池完整

### 4) `Vec[T]` 的值语义与运行时实现（临时方案）

Stage0/Stage1 v0 的 `Vec[T]` 在 C 后端中被表示为一个 by-value 的小结构体（含 `data/len/cap/elem_size`）。由于语言子集里大量值按位复制，`Vec` 也会被浅拷贝。

为避免浅拷贝 + `realloc/free` 触发悬垂指针/双重释放，本阶段采用了一个自举期的折中：

- `Vec` 扩容使用 `malloc + memcpy`，旧 buffer 不释放（故意泄漏）

这保证“复制后的 header”仍然指向有效 buffer，从而让 stage1 自举稳定。后续可用 move-only 语义、共享 buffer（RC）或真正的所有权/借用模型替换。

下一步（按依赖顺序）：

1. 工具链：把 Stage1 产出的 C 源码接入实际编译/链接，产出可执行文件（先 `main` 模块 + 最小 std）。
2. 诊断：为 Stage1 AST 增加最小 span/位置模型，提升 loader/typecheck/irgen 报错可用性。
3. 逐步扩展 Stage0 子集覆盖：更多类型、更多内建/stdlib（保持测试优先）。
