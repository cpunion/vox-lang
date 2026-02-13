# Vox Compiler Libraries（模块开放与分层）

目标：让 Vox 编译器能力像 Go 编译器库一样可复用，用户可以基于稳定 API 做 lint、代码生成、静态分析、定制构建工具。

## 1. 设计原则

1. 单一入口名：统一使用 `vox/*`。
2. 分层稳定性：公开层稳定，内部层允许重构。
3. 数据结构优先：先稳定 AST/Token/IR 这类数据模型，再稳定完整 driver。
4. 诊断一致性：所有层使用统一 span/diagnostic 语义。

## 2. 模块分层（当前与目标）

### 2.1 Stable（对外承诺兼容）

- `vox/ast`
- `vox/lex`（含 token/lexer）
- `vox/parse`
- `vox/manifest`
- `vox/ir`

适用场景：语法工具、格式化/重写、源码索引、依赖图分析、IR 检查工具。

### 2.2 Experimental（可用但不承诺稳定）

- `vox/typecheck`
- `vox/macroexpand`
- `vox/irgen`
- `vox/codegen`
- `vox/compile`
- `vox/loader`

适用场景：自定义编译流程、实验性后端、高级静态分析。

### 2.3 Internal（后续逐步下沉）

后续建议把仅供编译器自身使用的辅助逻辑收敛到 `vox/internal/*`，避免外部依赖内部实现细节。

## 3. 对齐 Go 风格的模块映射

对照 Go 生态（`go/ast`、`go/parser`、`go/token`、`go/types`）：

- `vox/ast` 对应 `go/ast`
- `vox/lex` 对应 `go/token + go/scanner`
- `vox/parse` 对应 `go/parser`
- `vox/typecheck` 对应 `go/types`（当前为 Experimental）
- `vox/manifest` 对应 `go/build` 的依赖/包清单侧能力

## 4. API 稳定性约定

1. Stable 层采用语义化版本兼容：
   - `v0.x`：允许小幅调整，但需记录迁移说明。
   - `v1.0` 后：禁止破坏性变更（除非 major 升级）。
2. Experimental 层变更可更快，但必须在 release notes 标注。
3. Internal 层不对外承诺兼容。

## 5. 推荐扩展模式

1. 语法工具：`vox/lex + vox/parse + vox/ast`。
2. 类型工具：在 `vox/typecheck` 稳定前，优先只读 AST/IR，减少对内部语义耦合。
3. 编译驱动：通过 CLI 或 `vox/compile`（Experimental）封装，避免直接依赖 `vox/codegen` 细节。

## 6. 近期落地任务

1. 为 Stable 层补最小示例（解析、AST 遍历、IR 打印）。
2. 给 Stable/Experimental 模块增加统一头注释（稳定性级别 + 迁移策略）。
3. 增加 `vox/*` 库级回归测试（作为对外 API 契约测试）。
