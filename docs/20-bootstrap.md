# 自举与阶段划分（Stage0 范围已定）

本章用于定义 Vox 的自举路线，以及 Stage0（宿主实现）的范围边界。

## 总体策略

1. **Stage0（Go）**：实现 Vox 的一个“可用于自举”的实用子集，重点是工程基础（包/多文件/测试/诊断），而不是语言全部特性。
2. **Stage1（Vox，自举）**：用 Vox 实现 Vox 编译器，目标是在功能覆盖上达到 Stage0，并在此基础上完善工程基座（测试/包管理/模块）以及编译链路（IR/后端/构建）。
3. **Stage2（工具链）**：在 Stage1 的编译链路稳定后，完善 fmt/lint/doc/LSP 等开发工具，并逐步纳入更多语言特性（comptime/宏等）。

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
- 模块：`import "x" [as a]` + `a.name(...)`（见 `docs/03-module-package.md` 与 `docs/14-syntax-details.md`）
- 可见性：默认私有 + `pub`（函数/结构体/枚举与结构体字段的最小子集）
- 控制流：`while` + `break/continue`
- 数据类型：`struct`（声明、字面量、字段读取/写入）
- IR/后端：`String` 字面量最小支持（IR `str` + C 后端）；`struct` 降低到 C `typedef struct`
- 数据类型：`enum` + `match`（Stage0 限制：variant payload 仅支持 0/1 个字段；后端降低为 `tag + union`）

Stage0 下一步（仍属“可包含”范围，优先级高）：
- 类型语法支持带路径的名义类型（例如 `a.Point` 出现在函数签名/字段类型中）

### 明确不包含（Stage0 非目标）

- 编译期计算：`comptime`（包括相关内建与执行器）
- 宏：`name!(...)`、`quote`/AST 类型
- trait/impl、泛型单态化（可后续再加）
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
