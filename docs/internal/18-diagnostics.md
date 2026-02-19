# 诊断与错误消息（v0）

本章定义 Vox v0 在当前主线编译器中的最小诊断输出约定，目标是：

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
- `col`：按 rune（Unicode code point）计数

## 2. 解析错误（parse/lex）

当前 lexer/parser 会产出 byte offset（`at`）。对外输出时应转换为 `line/col`（其中 `col` 为 rune 列）：

```text
src/main.vox:2:1: parse error: unexpected token: expected fn, got else
src/main.vox:3:5: lex error: unexpected char
```

`parse` 除文本渲染外，也提供结构化诊断元信息（从 `ParseError` 派生）：

- `kind`：`none/parse/lex`
- `code`：稳定错误码（`E_PARSE_0001`、`E_LEX_0001`）
- `message`：未渲染语义消息（不含 `file:line:col`）
- `rendered`：最终展示字符串（含 `file:line:col`）

## 3. 类型检查与后续阶段

当前实现的 typecheck/irgen 错误以字符串为主，**但当 AST 节点拥有 span 信息时**，必须输出精确的 `file:line:col`。

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
- trait/impl 的 associated type 声明已携带 `Span`，重复/非法绑定错误定位到具体 `type` 条目。
- const block 语句执行错误（`let/assign/assign field/if/while/break/continue`）优先使用语句 `Span`，不再回退 `file:1:1`。
- const 反射 intrinsic（如 `@same_type/@size_of/@is_*`）执行错误优先定位到调用表达式 `Span`，不再回退 `file:1:1`。
- macroexpand 轮次上限错误（`max rounds exceeded`）优先定位到首个宏调用点 `Span`，便于直接跳转问题源。
- typecheck/import/irgen 错误已附带稳定错误码后缀，当前包含：
  - `E_PARSE_0001`
  - `E_LEX_0001`
  - `E_TYPE_0001`
  - `E_IMPORT_0001`（通用 import 错误）
  - `E_IMPORT_0002`（unknown module import）
  - `E_IMPORT_0003`（ambiguous import）
  - `E_IMPORT_0004`（duplicate import alias）
  - `E_IMPORT_0005`（duplicate imported name）
  - `E_IMPORT_0006`（import name conflict）
  - `E_IMPORT_0007`（unknown imported name）
  - `E_IMPORT_0008`（ambiguous imported name）
  - `E_IMPORT_0009`（imported symbol is private）
  - `E_IRGEN_0001`

## 4. 错误分层（kind/code）

`typecheck` 对外保留文本渲染，同时在内部错误对象中附带结构化信息，便于测试与后续工具链扩展：

- `kind`：错误类别（`none/type/import`）
- `code`：稳定错误码（如 `E_TYPE_0001`）
- `message`：未渲染的语义消息
- `rendered`：最终展示字符串（仍含 `file:line:col` 前缀）

`compile` 层（`compile_files_to_c` / `compile_main_text_to_c`）同样暴露结构化错误字段，统一上游来源：

- `parse/lex` 失败：来自 `parse` 元信息
- `type/import` 失败：来自 `typecheck` 元信息
- `macroexpand` 失败：使用 compile 层独立分类（`kind = macroexpand`、`code = E_MACROEXPAND_0001`）
- `irgen`/`ir verify` 失败：使用 compile 层稳定 kind/code，并保留原始渲染文本
- `loader` 在文件聚合阶段也会保留并传递 `parse/lex` 的 `kind/code/message/rendered`

这样上层工具可在不解析错误字符串的前提下完成分流、统计和重试策略。

诊断升级（已落地）：

- AST/expr pool 节点已携带最小 span（`file/line/col`）。
- typecheck/irgen 关键错误路径已优先使用 span 渲染；缺失 span 时回退到 `1:1`。
