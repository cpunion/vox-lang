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
- parser：已支持 `import "pkg" as alias`、`pub`/`pub(crate)`/`pub(super)` + `struct`/`enum`/`trait`/`impl`/`fn`，以及语句 `let`/赋值/`if`/`while`/`break`/`continue`/`return`/表达式语句；表达式包含 member/call、struct literal、`match { pat => expr }`、`try { ... }`、后缀 `?` 与常见二元/一元运算（precedence climbing）。类型名已支持 `path`（`a.b.C`）与方括号泛型（`Vec[i32]`）；函数泛型参数已支持 trait bound 语法（`T: Eq + Show`）与 `where` 子句；trait 已支持 supertrait 头部语法（`trait B: A + C { ... }`）；`impl` 头部已支持 `impl[T: Bound] Trait for Type`。
- typecheck：已覆盖 Stage0 子集的主要路径（函数调用、member 调用、struct literal、enum ctor、match、Vec/String 最小内建），并支持泛型函数签名与泛型调用（可显式 `f[T](...)`，也可从参数/返回期望推导）；泛型名义类型 `struct[T]` / `enum[T]` 已支持按需实例化（`Name[Arg]`）。整数字面量/const/cast 对 `u64/usize` 已支持完整十进制范围 `0..18446744073709551615`（含 match pattern 与 const wrapping 运算），`const` 也已支持 `f32/f64` 的字面量/引用/负号、`int <-> float` 与 `f32 <-> f64` 转换、四则与比较/相等折叠、enum 常量（`.Variant` / `.Variant(...)`）、struct 常量（`Struct { ... }` + 字段读取）、`match`（当前 const 子集支持 `_`、bind、`true/false`、整数字面量、字符串字面量、enum pattern），以及最小块表达式（`{ let/let mut ...; assign ...; if ...; ...; tail }`，unit 上下文允许无 tail 分支块，支持 struct 字段赋值 `p.x = ...`）。`type Name = A: TA | B: TB` 形式的 union 在语义上按 enum 处理，已支持构造、match 与 const 场景。错误传播语法已接入：`expr?` 支持 `Result/Option` 两种容器（含返回类型一致性；`Result` 的 `Err` 先做直接兼容，再可选走 `std/prelude::Into` 转换），`try { ... }` 已具备块级传播边界与“成功值自动包装（Ok/Some）”。Trait MVP 已接入：支持 `trait`/`impl Trait for Type` 声明检查、impl 方法体类型检查、`Trait.method(x, ...)` 与 `alias.Trait.method(x, ...)` 的静态分发（按第一个参数类型选择 impl），并新增 `x.method(...)` 的 trait 语法糖静态分发；`impl[T: Bound]` 的泛型 impl 可参与上述静态分发（含 `Trait.method(x)` 与 `x.method()`）；trait 方法支持泛型参数与 trait bound（调用可显式给出类型实参），且在 `T: Trait` 约束下支持 `x.method(...)`（含 supertrait 方法）；支持 trait 默认方法（同模块/跨模块 trait 的默认实现都可被 `impl` 省略继承）；支持 supertrait 约束（`impl B for T` 需满足其父 trait `impl`），并支持 supertrait 前向引用（`trait B: A` 可先于 `trait A` 声明）；拒绝 supertrait 循环依赖。关联类型已支持“声明 + impl 绑定 + 完整性校验 + 类型位置投影”：可在 trait/impl 方法签名使用 `Self.Assoc`，并在泛型签名中使用 `T.Assoc`，且会对 `T.Assoc` 做 unknown/ambiguous 校验。泛型函数、泛型 impl 方法与泛型默认方法体会在定义阶段做类型检查（不再仅依赖实例化触发）。与此同时支持 `std/prelude` 的默认 trait 回退（调用与 `impl Trait for ...` 均可不显式 import）。泛型调用新增 trait bound 校验：实参类型必须存在对应 `impl`；并已加最小 coherence/orphan 规则（`impl` 要求 trait 或目标类型至少一个本地）。可见性规则已升级为 `pub`/`pub(crate)`/`pub(super)` 的完整语义检查（覆盖 import、类型引用、函数调用、字段访问、enum 构造与 const 引用）。`if` 表达式 lowering 的结果槽默认值现在可覆盖 `@range` 类型（使用区间下界作为确定性初值），避免 CFG 合流路径在 refined int 上报 unsupported。
- 数值类型：已支持 `f32/f64` 的字面量（含科学计数法与 `f32/f64` 后缀）、四则（`+ - * /`）、比较（`< <= > >=`）、相等（`== !=`）、`to_string()`，并支持 `as` 的 `int <-> float` 转换（其中 `float -> int` 为 checked cast），已打通到 IR 与 C 后端。
- IR v0：`compiler/stage1/src/ir/**` 已对齐 `docs/19-ir-spec.md`（TyPool、Value/Instr/Term、Program 结构与 formatter）。
- IRGen：`compiler/stage1/src/irgen/**` 已跑通 end-to-end（从 AST 生成 IR），并对泛型调用做单态化（worklist 生成可达的实例函数）；Trait MVP 的 impl 方法会被降低为内部函数符号并参与正常调用生成。
- codegen（C）：`compiler/stage1/src/codegen/**` 已支持 IR v0 -> 单文件 C 源码；并通过 `compiler/stage1/src/compile/**` 提供最小串联管线（parse -> typecheck -> irgen -> ir.verify -> codegen，用于端到端测试）。
- loader（in-memory）：`compiler/stage1/src/loader/**` 已支持按 `src/`/`tests/` 目录规则把多文件合并为模块（`src/*.vox -> main`，`src/<dir>/** -> <dir>`，`tests/** -> tests/...`），用于多模块端到端编译测试。
- stage1 CLI（最小）：`compiler/stage1/src/main.vox` 提供 `emit-c/build/build-pkg`，使用 `std/fs` 与 `std/process` 完成读写与调用系统 `cc`（用于自举前的工具链验证）。CLI 会自动注入 stage1 自带的 `src/std/**` 作为被编译包的本地 `src/std/**`；`build-pkg` 还会读取当前目录 `vox.toml` 并加载依赖 `src/**`（支持 `path`、`git`、以及本地 registry cache 的 `version` 解析，包含传递依赖）。另外 `emit-c/build/build-pkg` 支持 `--driver=user|tool`：`tool` 模式下生成的二进制不打印返回值，并把 `main() -> i32` 作为进程退出码返回（用于自举工具）。
- stage1 CLI（测试）：`compiler/stage1/src/main.vox` 还提供 `test-pkg`，发现并运行 `src/**/*_test.vox` 与 `tests/**/*.vox` 中的 `test_*`（行为对齐 stage0 的 `vox test`：单一测试二进制 + 每个测试单独进程运行）。
- stage0 集成测试已覆盖 Stage1 CLI 的关键路径：`emit-c`/`build`/`build-pkg`/`test-pkg`、`--driver=tool` 退出码语义、以及“包内 `src/std/**` 优先于嵌入 std 回退”的行为。

## 与自举相关的实现细节（当前实现）

### 1) 字符串转义与生成 C

- Stage1 lexer 仅负责识别 token 边界；parser 会对字符串字面量做最小的反转义（至少支持 `\\ \" \n \r \t`）。
- codegen 输出到 C 字符串字面量时，会对内容做 C 级别转义（例如把换行字符输出为 `\\n`），避免生成的 `.c` 文件被“字面量换行”破坏。

### 2) 名义类型相等（Stage0/Stage1 v0 约束）

为保持范围可控，Stage0/Stage1 对 `==/!=` 的支持与 lowering 有明确限制（详见 `docs/14-syntax-details.md`）：

- `bool/<int>/f32/f64/String`：直接比较
- `struct`：逐字段比较（字段类型需可比较）
- `enum`：先比较 tag，再对 payload 逐字段比较（字段类型需可比较）

后端会为名义类型生成辅助函数（`vox_struct_eq_*` / `vox_enum_eq_*`），并在 `cmp_eq/cmp_ne` 处调用。

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

当前补充：

1. Const 泛型默认值：已支持函数/trait 方法声明默认 const 参数，并支持调用端省略/覆盖。
2. 包管理：`vox.toml` 依赖项支持 `path` 与 `version` 字段解析；`build-pkg`/`test-pkg` 会生成 `vox.lock`。
3. 类型反射 intrinsic：已支持 `@size_of/@align_of/@type/@type_name/@field_count/@field_name/@field_type/@field_type_id/@same_type/@assignable_to/@castable_to/@eq_comparable_with/@ordered_with/@same_layout/@bitcastable` 以及 `@is_integer/@is_signed_int/@is_unsigned_int/@is_float/@is_bool/@is_string/@is_struct/@is_enum/@is_vec/@is_range/@is_eq_comparable/@is_ordered/@is_unit/@is_numeric/@is_zero_sized`，并在 const 与 IR lowering 阶段常量折叠。
