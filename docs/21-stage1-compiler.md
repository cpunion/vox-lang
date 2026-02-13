# Stage1 编译器（Vox 实现，冻结维护线）

本章描述 `compiler/stage1` 的工程结构与维护边界。当前仓库策略是：Stage1 作为稳定 bootstrap 编译器冻结维护，不再承载新增语言能力；新增能力在 Stage2 演进。

## 目录约定

- `compiler/stage1/src/main.vox`：编译器入口（暂为占位）。
- `compiler/stage1/src/std/**`：Stage1 的标准库源码（由 Stage0 注入，用于 Stage0/Stage1 共用最小工具）。
- `compiler/stage1/src/**/**_test.vox`：Stage1 自身的单元测试，与实现代码同目录同包（由 Stage0 的 `vox test` 运行）。
- `compiler/stage1/src/compile/**`：Stage1 的最小“串联管线”（parse -> typecheck -> irgen -> codegen）入口，便于在无 IO 的前提下做端到端测试。

与 Stage2 的关系（当前仓库约定）：

- `compiler/stage2` 以 `compiler/stage1` 当前实现为基线复制维护。
- Stage2 编译器由 Stage1 编译生成（已在 Stage0 集成测试覆盖 `stage1 -> stage2` 引导链路）。
- 这条链路用于承接“Stage0 无法直接承载”的语言演进（例如更激进的泛型标准库抽象）。

## 当前维护边界

1. 保持 `stage1 -> stage2` 引导链路稳定（优先级最高）。
2. 仅修复回归、崩溃、错误诊断与明显性能退化。
3. 不在 Stage1 新增语法/类型系统能力；相关开发在 `compiler/stage2` 进行。

说明：

- Stage1 lexer 的 `Token` 不直接携带 `lexeme`，而是携带 `[start,end)` 的 byte offset。解析器需要通过 `source.slice(start, end)` 拉取 token 文本。
- Stage0/Stage1/Stage2 当前仍使用 `String.slice(start, end) -> String` 作为子串基础操作；切片方向通过标准库 `std/string::StrView`（`owner + lo/hi`）补齐。Stage2 已支持 `&T` / `&mut T` / `&'static T` / `&'static mut T` 语法，但当前在类型层仍按 `T` 过渡语义处理；命名 lifetime（`&'a T`）由 parser 直接拒绝；另外裸 `str` 已禁用，需使用 `String` 或 `&str`/`&'static str`。

当前进度（实现状态以代码为准）：

- lexer：已覆盖常用关键字/标点、字符串/整数、注释与错误定位（byte offset）。
- parser：已支持 `import "pkg" as alias`、`pub`/`pub(crate)`/`pub(super)` + `struct`/`enum`/`trait`/`impl`/`fn`，以及语句 `let`/赋值/`if`/`while`/`break`/`continue`/`return`/表达式语句；表达式包含 member/call、struct literal、`match { pat => expr }`、`try { ... }`、后缀 `?` 与常见二元/一元运算（precedence climbing）。类型名已支持 `path`（`a.b.C`）与方括号泛型（`Vec[i32]`）；函数泛型参数已支持 trait bound 语法（`T: Eq + Show`）与 `where` 子句；trait 已支持 supertrait 头部语法（`trait B: A + C { ... }`）；`impl` 头部已支持 `impl[T: Bound] Trait for Type`。字符串字面量新增 `"""..."""` 多行形式，按“首行换行可去除 + 最小公共缩进去除（仅空格）”规则 unindent。
- typecheck：已覆盖 Stage0 子集的主要路径（函数调用、member 调用、struct literal、enum ctor、match、Vec/String 最小内建），并支持泛型函数签名与泛型调用（可显式 `f[T](...)`，也可从参数/返回期望推导）；泛型名义类型 `struct[T]` / `enum[T]` 已支持按需实例化（`Name[Arg]`）。整数字面量/const/cast 对 `u64/usize` 已支持完整十进制范围 `0..18446744073709551615`（含 match pattern 与 const wrapping 运算），`const` 也已支持 `f32/f64` 的字面量/引用/负号、`int <-> float` 与 `f32 <-> f64` 转换、四则与比较/相等折叠、enum 常量（`.Variant` / `.Variant(...)`）、struct 常量（`Struct { ... }` + 字段读取）、`match`（当前 const 子集支持 `_`、bind、`true/false`、整数字面量、字符串字面量、enum pattern），以及最小块表达式（`{ let/let mut ...; assign ...; if ...; ...; tail }`，unit 上下文允许无 tail 分支块，支持 struct 字段赋值 `p.x = ...`）。`type Name = A: TA | B: TB` 形式的 union 在语义上按 enum 处理，已支持构造、match 与 const 场景。错误传播语法已接入：`expr?` 支持 `Result/Option` 两种容器（含返回类型一致性；`Result` 的 `Err` 先做直接兼容，再可选走 `std/prelude::Into` 转换），`try { ... }` 已具备块级传播边界与“成功值自动包装（Ok/Some）”。Trait MVP 已接入：支持 `trait`/`impl Trait for Type` 声明检查、impl 方法体类型检查、`Trait.method(x, ...)` 与 `alias.Trait.method(x, ...)` 的静态分发（按第一个参数类型选择 impl），并新增 `x.method(...)` 的 trait 语法糖静态分发；`impl[T: Bound]` 的泛型 impl 可参与上述静态分发（含 `Trait.method(x)` 与 `x.method()`）；trait 方法支持泛型参数与 trait bound（调用可显式给出类型实参），且在 `T: Trait` 约束下支持 `x.method(...)`（含 supertrait 方法）；支持 trait 默认方法（同模块/跨模块 trait 的默认实现都可被 `impl` 省略继承）；支持 supertrait 约束（`impl B for T` 需满足其父 trait `impl`），并支持 supertrait 前向引用（`trait B: A` 可先于 `trait A` 声明）；拒绝 supertrait 循环依赖。关联类型已支持“声明 + impl 绑定 + 完整性校验 + 类型位置投影”：可在 trait/impl 方法签名使用 `Self.Assoc`，并在泛型签名中使用 `T.Assoc`，且会对 `T.Assoc` 做 unknown/ambiguous 校验。泛型函数、泛型 impl 方法与泛型默认方法体会在定义阶段做类型检查（不再仅依赖实例化触发）。与此同时支持 `std/prelude` 的默认 trait 回退（调用与 `impl Trait for ...` 均可不显式 import）。泛型调用新增 trait bound 校验：实参类型必须存在对应 `impl`；并已加最小 coherence/orphan 规则（`impl` 要求 trait 或目标类型至少一个本地）。可见性规则已升级为 `pub`/`pub(crate)`/`pub(super)` 的完整语义检查（覆盖 import、类型引用、函数调用、字段访问、enum 构造与 const 引用）。`if` 表达式 lowering 的结果槽默认值现在可覆盖 `@range` 类型（使用区间下界作为确定性初值），避免 CFG 合流路径在 refined int 上报 unsupported。
- 数值类型：已支持 `f32/f64` 的字面量（含科学计数法与 `f32/f64` 后缀）、四则（`+ - * /`）、比较（`< <= > >=`）、相等（`== !=`）、`to_string()`，并支持 `as` 的 `int <-> float` 转换（其中 `float -> int` 为 checked cast），已打通到 IR 与 C 后端。
- IR v0：`compiler/stage1/src/ir/**` 已对齐 `docs/19-ir-spec.md`（TyPool、Value/Instr/Term、Program 结构与 formatter）。
- IRGen：`compiler/stage1/src/irgen/**` 已跑通 end-to-end（从 AST 生成 IR），并对泛型调用做单态化（worklist 生成可达的实例函数）；Trait MVP 的 impl 方法会被降低为内部函数符号并参与正常调用生成。
- codegen（C）：`compiler/stage1/src/codegen/**` 已支持 IR v0 -> 单文件 C 源码；并通过 `compiler/stage1/src/compile/**` 提供最小串联管线（parse -> typecheck -> irgen -> ir.verify -> codegen，用于端到端测试）。
- loader（in-memory）：`compiler/stage1/src/loader/**` 已支持按 `src/`/`tests/` 目录规则把多文件合并为模块（`src/*.vox -> main`，`src/<dir>/** -> <dir>`，`tests/** -> tests/...`），用于多模块端到端编译测试。
- stage1 CLI（最小）：`compiler/stage1/src/main.vox` 提供 `emit-c/build/build-pkg`，使用 `std/fs` 与 `std/process` 完成读写与调用系统 C 编译器（优先读取 `CC` 环境变量；未设置时回退 `cc/gcc`，用于自举前的工具链验证）。CLI 会自动注入 stage1 自带的 `src/std/**` 作为被编译包的本地 `src/std/**`；`build-pkg` 还会读取当前目录 `vox.toml` 并加载依赖 `src/**`（支持 `path`、`git`、以及本地 registry cache 的 `version` 解析，包含传递依赖）。另外 `emit-c/build/build-pkg` 支持 `--driver=user|tool`：`tool` 模式下生成的二进制不打印返回值，并把 `main() -> i32` 作为进程退出码返回（用于自举工具）。
- stage1 CLI（测试）：`compiler/stage1/src/main.vox` 还提供 `test-pkg`，发现并运行 `src/**/*_test.vox` 与 `tests/**/*.vox` 中的 `test_*`（行为对齐 stage0 的 `vox test`：单一测试二进制 + 每个测试单独进程运行）。
- stage0 集成测试已覆盖 Stage1 CLI 的关键路径：`emit-c`/`build`/`build-pkg`/`test-pkg`、`--driver=tool` 退出码语义、以及“包内 `src/std/**` 优先于嵌入 std 回退”的行为。
- stage0 自举门禁已覆盖 `stage1 -> stage2 -> stage2 test-pkg`，确保 Stage2 主线迭代不会悄悄破坏自举链路。

## 与自举相关的实现细节（当前实现）

### 1) 字符串转义与生成 C

- Stage1 lexer 仅负责识别 token 边界；parser 会对字符串字面量做最小反转义（至少支持 `\\ \" \n \r \t`），并对 `"""..."""` 应用 unindent 规则（TAB 缩进报错）。
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

### 4) `Vec[T]` 的运行时内存模型（当前收敛方案）

Stage1/Stage2 的 C runtime 已从“扩容链式泄漏”收敛到共享存储头模型：

- `vox_vec` 现在是 `{ h: vox_vec_data*, len }`，其中 `vox_vec_data` 持有 `data/cap/elem_size`。
- 扩容改为 `realloc(h->data, ...)`，不再在每次 grow 保留旧 buffer。
- 由于浅拷贝仍可能出现，`h` 作为间接层可避免 grow 后旧副本悬垂指针（UAF）。

当前边界（stage1 冻结线的折中语义）：

- 仍未引入通用 drop/release；进程结束前活跃容器及其缓冲区保持有效（不做完整生命周期回收）。
- 复制后的值共享同一 backing storage（`h`），但持有各自的 `len`；这是为自举稳定与实现复杂度做的折中，后续如引入严格所有权/移动语义可再收敛为更强一致模型。

Stage2 补充（在不引入语言级 drop 的前提下）：

- C runtime 增加了“运行时分配跟踪 + `atexit` 清理”机制：`vox_vec` backing storage、字符串运行时结果（`slice/concat/to_string` 等）和部分句柄分配会注册到跟踪表，并在进程退出时统一释放。
- 目录遍历/路径辅助运行时（`mkdir_p`、`walk_vox_files`、`path_join2`）已统一使用 `vox_rt_malloc` 跟踪分配，不再混用裸 `malloc/free`。
- 运行时新增 `vox_rt_free`（配合跟踪表移除）用于“可提前释放”的临时分配；`mkdir_p` 与目录遍历中不逃逸的路径缓冲已在使用后立即释放，降低长流程工具命令的峰值内存。
- 该机制只解决“进程生命周期内累计泄漏”问题，不改变值语义；容器共享 backing storage、浅拷贝行为与当前 type system 约束保持不变。
- `std/sync` 新增显式释放路径：`mutex_drop/atomic_drop`（对应低层 `__mutex_*_drop/__atomic_*_drop`），可在长流程工具中提前释放句柄分配；不改变当前值语义与浅拷贝语义。
- `vox_rt_free` 改为“仅释放 tracked 指针”：若指针未被跟踪或已释放则直接忽略，避免在当前值拷贝/共享句柄语义下重复显式 drop 导致 double-free。

当前补充：

1. Const 泛型默认值：已支持函数/trait 方法声明默认 const 参数，并支持调用端省略/覆盖。
2. 包管理：`vox.toml` 依赖项支持 `path` 与 `version` 字段解析；`build-pkg`/`test-pkg` 会生成 `vox.lock`。
   - 现补充：`vox.lock` 记录依赖 `digest`（`vox.toml` + `src/**/*.vox` 摘要），后续构建会做一致性校验，不匹配直接失败，保证可复现。
   - lock mismatch 诊断已细化到字段级（如 `dependency mismatch: dep field digest expected=... actual=...`），并在 `build-pkg`/`test-pkg` 路径输出一致修复提示。
3. 类型反射 intrinsic：已支持 `@size_of/@align_of/@type/@type_name/@field_count/@field_name/@field_type/@field_type_id/@same_type/@assignable_to/@castable_to/@eq_comparable_with/@ordered_with/@same_layout/@bitcastable` 以及 `@is_integer/@is_signed_int/@is_unsigned_int/@is_float/@is_bool/@is_string/@is_struct/@is_enum/@is_vec/@is_range/@is_eq_comparable/@is_ordered/@is_unit/@is_numeric/@is_zero_sized`，并在 const 与 IR lowering 阶段常量折叠。
4. stdlib 基座：`std/sync` 已统一为泛型句柄 API（`Mutex[T]/Atomic[T]`，当前由 `SyncScalar` 约束覆盖 `i32/i64`）；`std/io` 网络最小 TCP API（`net_connect/net_send/net_recv/net_close`）在解释器与 C 后端可用。
5. 诊断定位：AST 顶层声明已携带 `Span`，typecheck/irgen 的声明级错误与 `missing return` 已优先输出真实 `file:line:col`。
6. 自举兼容约束（当前实例）：`std/collections/map.vox` 在 Stage2 源码中采用 bootstrap-safe 实现（避免 `Vec.set/remove/insert` 与局部 `Map[K,V]` let 注解），以保证 `stage1 -> stage2` 链路稳定。退出条件：Stage1 补齐对应语义后，切回更直接实现并删除兼容层，同时保持 `make test-stage2-tests` 绿灯。
