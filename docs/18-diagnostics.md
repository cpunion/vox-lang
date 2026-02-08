# 诊断与错误消息（v0）

本章定义 Vox v0（Stage0/Stage1 自举期）的最小诊断输出约定，目标是：

- 错误消息可用于定位：至少包含 `file:line:col`
- 输出稳定：便于测试用例断言与回归
- 实现成本可控：先支持 byte offset 到 `line/col` 的映射，不追求完整的源代码片段高亮

## 1. 基本格式

最小格式：

```text
<file>:<line>:<col>: <message>
```

其中：

- `file`：源码路径（VFS 虚拟路径也可，如依赖包文件 `dep/src/x.vox`）
- `line/col`：1-based
- `col`：按 UTF-8 byte 计数（Stage1 v0 约定；后续可升级为 rune/column）

## 2. 解析错误（parse/lex）

Stage1 v0 的 lexer/parser 会产出 byte offset（`at`）。对外输出时应转换为 `line/col`：

```text
src/main.vox:2:1: parse error: unexpected token: expected fn, got else
src/main.vox:3:5: lex error: unexpected char
```

## 3. 类型检查与后续阶段

Stage1 v0 的 typecheck/irgen 错误以字符串为主，**但当 AST 节点拥有 span 信息时**，必须输出精确的 `file:line:col`。

在 AST/Span 尚未完整接入之前，**仍然要求输出遵循本章基本格式**；当无法确定精确位置时：

- `line/col` 统一使用 `1:1` 作为兜底
- `file` 必须尽可能准确（例如函数所属的源文件、import 所在文件）

后续计划（Stage1 诊断升级）：

- AST/expr pool 节点携带最小 span（至少包含 `file/line/col`；byte offset 可选）
- typecheck/irgen 错误携带 span，并按本章格式渲染；缺失 span 时回退到 `1:1`
