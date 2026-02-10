# 自举与阶段划分（Stage0 范围已定）

本章用于定义 Vox 的自举路线，以及 Stage0（宿主实现）的范围边界。

## 总体策略

1. **Stage0（Go）**：实现 Vox 的一个“可用于自举”的实用子集，重点是工程基础（包/多文件/测试/诊断），而不是语言全部特性。
2. **Stage1（Vox，自举）**：用 Vox 实现 Vox 编译器，目标是在功能覆盖上达到 Stage0，并在此基础上完善工程基座（测试/包管理/模块）以及编译链路（IR/后端/构建）。
3. **Stage2（Vox，Stage1 产出）**：以 Stage1 为基线复制并演进的下一代编译器线，由 Stage1 编译生成。该阶段用于承载超出 Stage0 约束的语言能力（例如更激进的泛型/标准库抽象演进）。
4. **Stage3（工具链）**：在 Stage2 编译链路稳定后，完善 fmt/lint/doc/LSP 等开发工具，并逐步纳入 comptime/宏等更高阶特性。

## 当前开发策略（冻结与演进）

- `Stage0`：进入冻结维护期，仅接受回归修复、稳定性修复、性能回归修复与自举门禁相关修复；不再新增语言能力。
- `Stage1`：进入冻结维护期，作为 `Stage2` 的稳定 bootstrap 编译器；不再承载新增语言特性（仅修复会阻断 `stage1 -> stage2` 链路的问题）。
- `Stage2`：作为唯一的语言能力演进主线（类型系统、泛型/trait 增强、标准库扩展、工具链能力前移等都在此推进）。

冻结并不意味着不测试：`Stage0/Stage1` 仍保持强门禁，确保 `Stage2` 迭代不会破坏 bootstrap 链路与兼容性基线。

推荐日常命令：

- `make test-active`：冻结维护期的主线门禁（`stage0` 单测 + `stage1 -> stage2` 引导链路 + `stage2` 自测套件）。

## Stage0 范围（已定）

Stage0 的目标是“能写编译器”，但保持范围可控，不引入会显著放大实现复杂度的特性。

### 必须包含（工程基础）

- 包：`vox.toml`（最小可用子集，至少支持 `[package]` 与 path 依赖的校验）
- 多文件：一个包可包含多个 `.vox` 源文件（`src/` 递归）
- 测试：`vox test`（最小可用测试发现与运行）
- 诊断：文件/行列定位、稳定错误信息

### 必须包含（语言子集）

- 基本类型：`i32`、`i64`、`bool`、`String`、`()`
- 函数：`fn`、参数/返回、调用、递归
- 绑定与可变：`let`、`let mut`、赋值、`return`
- 控制流：`if/else`
- 表达式：字面量、变量、算术/比较/逻辑运算

### 可以包含（避免 Stage1 代码啰嗦）

这些能力对“实现编译器”的代码体量与可读性影响很大，但仍然可控：

- `struct` / `enum` + `match`
- `Vec[T]` 等基础容器（可先作为内建类型/宿主库）

当前 Stage0 已实现（增量更新）：

- 测试：Go 风格 `*_test.vox` + `test_*` 发现（见 `docs/12-testing-framework.md`）
- 标准库注入：默认注入内置 std；若包内存在 `src/std/**`，则自动切换为“仅本地 std”（不叠加注入）
- 模块：`import "x" [as a]` + `a.name(...)`（见 `docs/03-module-package.md` 与 `docs/14-syntax-details.md`）
- 可见性：`pub` / `pub(crate)` / `pub(super)`（覆盖 import、类型引用、函数调用、字段访问、enum 构造与 const 引用）
- 类型路径：类型位置支持 `a.Type`（用于跨模块/包引用名义类型）
- 命名导入：`import { name } from "path"`（最小子集；减少 `a.name(...)` 噪声）
- 控制流：`while` + `break/continue`
- 控制流：`if` 支持表达式形式 `if cond { ... } else { ... }`（分支为表达式块，可包含语句与可选 tail expr）
- 数据类型：`struct`（声明、字面量、字段读取/写入）
- 容器：`Vec[T]`（`Vec()` 构造 + `push/len/get` 的最小子集；先作为内建类型）
- IR/后端：`String` 字面量最小支持（IR `str` + C 后端）；`struct` 降低到 C `typedef struct`
- 数据类型：`enum` + `match`（variant payload 支持 0..N 个字段；后端降低为 `tag + union`）

Stage0 下一步（仍属“可包含”范围，优先级高）：
- 继续扩大 trait/泛型/const-eval 子集覆盖，减少 Stage1 自举阶段的样板代码

### 明确不包含（Stage0 非目标）

- 编译期计算：`comptime`（包括相关内建与执行器）
- 宏：`name!(...)`、`quote`/AST 类型
- trait/impl（可后续再加）
- async/await
- effect/资源系统

## Stage1 覆盖策略

Stage1 的首要目标是覆盖 Stage0 的能力与工具链行为，避免出现“Stage0 能跑但 Stage1 还写不出来”的断层。

建议优先顺序：

1. 解析/多文件加载/包系统
2. 类型检查（覆盖 Stage0 子集）
3. `vox test` 一致性与回归集（让 compiler 工程可持续演进）
4. IR + 后端（先能产出可执行/可链接产物；优化可后置）
5. 构建系统/包管理完善（依赖解析、锁文件、增量构建等逐步引入）

## 自举验证（已落地的回归测试）

为避免 Stage1 演进过程中悄悄破坏“自举链路”，仓库内维护了端到端集成测试（Stage0 Go 测试）：

- 测试文件：`compiler/stage0/cmd/vox/stage1_integration_test.go`
- 覆盖场景（简化描述）：
  1. Stage0 编译 Stage1 编译器 A（`vox_stage1`）
  2. 用 Stage1 A 自举编译 Stage1 编译器 B（输出到 `compiler/stage1/target/debug/`）
  3. 用 Stage1 B 编译一个临时包（包含 `vox.toml` 的 path 依赖 + `import "std/prelude"`），并运行产物验证结果

运行方式：

```bash
make test
```

实现约束（当前实现）：

- Stage1 CLI 在运行时会根据 `std/process.exe_path()` 推导 Stage1 根目录，从其 `src/std/**` 加载标准库源码。
- 因此 self-build 时 Stage1 B 的输出路径需要位于 `compiler/stage1/target/debug/`（测试也固定了这一点）。

补充（Stage2 引导）：

- 仓库已增加 `compiler/stage2`（以 `compiler/stage1` 当前实现为基线复制）。
- 自举门禁新增 `TestStage1BuildsStage2AndBuildsPackage`：验证 `stage1 -> stage2` 编译链路可用。
- 自举门禁新增 `TestStage1BuildsStage2AndRunsStage2Tests`：验证 `stage2` 工具可运行 `test-pkg` 并通过其自身测试集。
- `stage2 test-pkg` 当前支持最小选择参数：`--filter=<text>` 与 `--list`（用于缩小迭代验证范围）。
- 可通过 `make test-stage2-selfhost`（仅引导链路）与 `make test-stage2-tests`（stage2 测试套件）单独执行（依赖 `VOX_RUN_SELFHOST_TESTS=1`）。
