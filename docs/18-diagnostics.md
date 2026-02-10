# 诊断与错误消息（v0）

本章定义 Vox v0（Stage0/Stage1 自举期）的最小诊断输出约定，目标是：

- 错误消息可用于定位：至少包含 `file:line:col`
- 输出稳定：便于测试用例断言与回归
- 实现成本可控：支持 byte offset 到 `line/rune-col` 的映射，不追求完整的源代码片段高亮

## 1. 基本格式

最小格式：

```text
<file>:<line>:<col>: <message>
```

其中：

- `file`：源码路径（VFS 虚拟路径也可，如依赖包文件 `dep/src/x.vox`）
- `line/col`：1-based
- `col`：
  - Stage0：按 rune（Unicode code point）计数
  - Stage1：按 rune（Unicode code point）计数

## 2. 解析错误（parse/lex）

Stage1 的 lexer/parser 会产出 byte offset（`at`）。对外输出时应转换为 `line/col`（其中 `col` 为 rune 列）：

```text
src/main.vox:2:1: parse error: unexpected token: expected fn, got else
src/main.vox:3:5: lex error: unexpected char
```

## 3. 类型检查与后续阶段

Stage1 v0 的 typecheck/irgen 错误以字符串为主，**但当 AST 节点拥有 span 信息时**，必须输出精确的 `file:line:col`。

建议优先级：

- 表达式错误：使用表达式自身的 span（例如 call/field/match 等）
- 语句错误：使用语句起始 token 的 span
- 顶层声明错误：若暂时没有 span，则回退到 `file:1:1`

在 AST/Span 尚未完整接入之前，**仍然要求输出遵循本章基本格式**；当无法确定精确位置时：

- `line/col` 统一使用 `1:1` 作为兜底
- `file` 必须尽可能准确（例如函数所属的源文件、import 所在文件）

当前已接入（2026-02）：

- 顶层声明（`type/const/struct/enum/trait/impl/fn`）已携带最小 `Span`。
- collect/typecheck 的一批声明级错误（如 duplicate/reserved/bound/impl 校验）已优先使用声明 `Span`。
- irgen 的 `missing return` 已使用函数声明 `Span`，不再固定 `file:1:1`。
- import 相关错误（重复 alias、命名导入冲突、unknown/ambiguous import）已优先使用 `import` 声明/名称位置。
- `comptime where` 声明期错误（unknown rhs/unknown type param/default 违规等）已优先使用声明 `Span`。
- supertrait cycle 报错已定位到 trait 声明位置（不再回退 `file:1:1`）。
- trait 方法声明已携带 `Span`，其方法级 `where/comptime where` 错误定位到方法行而非 trait 头。

后续计划（Stage1 诊断升级）：

- AST/expr pool 节点携带最小 span（至少包含 `file/line/col`；byte offset 可选）
- typecheck/irgen 错误携带 span，并按本章格式渲染；缺失 span 时回退到 `1:1`
