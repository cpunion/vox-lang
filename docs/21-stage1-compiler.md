# Stage1 编译器（Vox 实现，草案）

本章描述 `compiler/stage1` 的工程结构与近期开发顺序。Stage1 的目标是在 Vox 语言子集内实现 Vox 编译器，并最终替代 Stage0（Go）。

## 目录约定

- `compiler/stage1/src/main.vox`：编译器入口（暂为占位）。
- `compiler/stage1/src/std/**`：Stage1 的标准库源码（由 Stage0 注入，用于 Stage0/Stage1 共用最小工具）。
- `compiler/stage1/src/stage1_tests/**`：Stage1 自身的测试（由 Stage0 的 `vox test` 运行）。

## 近期顺序（可迭代）

1. 词法（lexer）：把 `String` 解析为 token 流，包含位置（byte offset）。
2. 语法（parser）：从 token 流构建 AST（使用 arena/index 建模递归结构）。
3. 类型检查：覆盖 Stage0 子集。
4. IR v0：对齐 `docs/19-ir-spec.md`，先跑通从 AST 到 IR。
5. 后端：先复用 stage0 的 C 后端策略（tagged union、by-value struct），能产出可执行文件。

说明：在 Stage0 子集内暂时没有 substring/切片，因此 lexer token 先用 `start/end`（byte offset）引用源文本，不直接携带 `lexeme` 字符串。

