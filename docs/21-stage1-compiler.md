# Stage1 编译器（Vox 实现，草案）

本章描述 `compiler/stage1` 的工程结构与近期开发顺序。Stage1 的目标是在 Vox 语言子集内实现 Vox 编译器，并最终替代 Stage0（Go）。

## 目录约定

- `compiler/stage1/src/main.vox`：编译器入口（暂为占位）。
- `compiler/stage1/src/std/**`：Stage1 的标准库源码（由 Stage0 注入，用于 Stage0/Stage1 共用最小工具）。
- `compiler/stage1/src/**/**_test.vox`：Stage1 自身的单元测试，与实现代码同目录同包（由 Stage0 的 `vox test` 运行）。

## 近期顺序（可迭代）

1. 词法（lexer）：把 `String` 解析为 token 流，包含位置（byte offset）。
2. 语法（parser）：从 token 流构建 AST（使用 arena/index 建模递归结构）。
3. 类型检查：覆盖 Stage0 子集。
4. IR v0：对齐 `docs/19-ir-spec.md`，先跑通从 AST 到 IR。
5. 后端：先复用 stage0 的 C 后端策略（tagged union、by-value struct），能产出可执行文件。

说明：

- Stage1 lexer 的 `Token` 不直接携带 `lexeme`，而是携带 `[start,end)` 的 byte offset。解析器需要通过 `source.slice(start, end)` 拉取 token 文本。
- Stage0 已提供 `String.slice(start, end) -> String` 作为过渡能力；其内存行为在 Stage0/C 后端下会产生分配（临时泄漏可接受，后续再用切片/真实字符串模型替换）。

当前进度（实现状态以代码为准）：

- lexer：已覆盖常用关键字/标点、字符串/整数、注释与错误定位（byte offset）。
- parser：已支持 `import "pkg" as alias`、`pub struct`/`enum`/`fn`，以及语句 `let`/赋值/`if`/`while`/`break`/`continue`/`return`/表达式语句；表达式包含 member/call 与常见二元/一元运算（precedence climbing）。
