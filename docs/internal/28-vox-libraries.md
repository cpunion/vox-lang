# Vox Compiler Libraries（模块开放与分层）

目标：让 Vox 编译器能力像 Go 编译器库一样可复用，用户可以基于稳定 API 做 lint、代码生成、静态分析、定制构建工具。

## 1. 设计原则

1. 单一入口名：统一使用 `vox/*`。
2. 分层稳定性：公开层稳定，内部层允许重构。
3. 数据结构优先：先稳定位置系统/AST/Token/IR，再稳定完整 driver。
4. 诊断一致性：所有层使用统一 span/diagnostic 语义。

## 2. 模块分层（当前与目标）

### 2.1 Stable（对外承诺兼容）

- `vox/token`
- `vox/ast`
- `vox/lex`（scanner/tokenize）
- `vox/parse`
- `vox/manifest`
- `vox/ir`

适用场景：语法工具、格式化/重写、源码索引、依赖图分析、IR 检查工具。

### 2.2 Experimental（可用但不承诺稳定）

- `vox/types`
- `vox/typecheck`
- `vox/macroexpand`
- `vox/irgen`
- `vox/codegen`
- `vox/compile`
- `vox/loader`
- `vox/list`

适用场景：自定义编译流程、实验性后端、高级静态分析。

### 2.3 Internal（后续逐步下沉）

后续把仅供编译器自身使用的辅助逻辑收敛到 `vox/internal/*`，避免外部依赖内部实现细节。

## 3. 对齐 Go 风格的模块映射

对照 Go 生态（`go/token`、`go/scanner`、`go/ast`、`go/parser`、`go/types`、`go list`）：

- `vox/token` 对应 `go/token`
- `vox/lex` 对应 `go/scanner`
- `vox/ast` 对应 `go/ast`
- `vox/parse` 对应 `go/parser`
- `vox/types` 对应 `go/types`（当前为 Experimental）
- `vox/typecheck` 为 `vox/types` 的后端实现细节（Experimental）
- `vox/manifest` + `vox/list` 对应 `go/build` + `go list`

## 4. API 稳定性约定

1. Stable 层采用语义化版本兼容：
   - `v0.x`：允许小幅调整，但需记录迁移说明。
   - `v1.0` 后：禁止破坏性变更（除非 major 升级）。
2. Experimental 层变更可更快，但必须在 release notes 标注。
3. Internal 层不对外承诺兼容。

## 5. 最小示例

### 5.1 `vox/token`

```vox
import "vox/token" as tok

fn pos_text() -> String {
  let ar: tok.AddFileResult = tok.file_set_add_file_from_text(tok.file_set(), "src/main.vox", "a\nb");
  let ps: tok.Pos = tok.file_set_pos(ar.fset, ar.file_idx, tok.off_from_raw(2));
  if !tok.pos_is_valid(ps) { return "<invalid>"; }
  let p: tok.Position = tok.file_set_position(ar.fset, ps);
  return tok.position_string(p); // src/main.vox:2:1
}
```

### 5.2 `vox/lex`

```vox
import "vox/lex" as lex

fn first_token_kind(src: String) -> String {
  let r: lex.LexResult = lex.lex_text(src);
  if match r.err { lex.LexError.None => false, _ => true } { return "err"; }
  return lex.token_kind_name(r.tokens.get(0).kind);
}
```

### 5.3 `vox/parse`

```vox
import "vox/parse" as p

fn fn_count(src: String) -> i32 {
  let r: p.ParseResult = p.parse_text(src);
  if match r.err { p.ParseError.None => false, _ => true } { return -1; }
  return r.prog.funcs.len();
}
```

### 5.4 `vox/manifest`

```vox
import "vox/manifest" as mf

fn dep_count(text: String) -> i32 {
  let r: mf.ParseResult = mf.parse(text);
  if !r.ok { return -1; }
  return r.m.deps.len();
}
```

### 5.5 `vox/ir`

```vox
import "vox/ir" as ir

fn empty_ir_text() -> String {
  let add: ir.AddTyResult = ir.ty_pool_add(ir.ty_pool(), ir.ty_i32());
  let prog: ir.Program = ir.program(add.pool);
  return ir.format_program(prog);
}
```

### 5.6 `vox/ast`

```vox
import "vox/ast" as ast

fn default_span_line() -> i32 {
  let sp: ast.Span = ast.span0();
  return sp.line;
}
```

### 5.7 `vox/types`

```vox
import "vox/parse" as p
import "vox/typecheck" as tc
import "vox/types" as tys

fn types_smoke() -> bool {
  let pr: p.ParseResult = p.parse_text_with_path("src/main.vox", "fn main() -> i32 { return 0; }");
  if match pr.err { p.ParseError.None => false, _ => true } { return false; }

  let mut w: tc.World = tc.world();
  w = tc.world_add(w, "main", pr.prog);

  let r: tys.CheckResult = tys.check_world(w);
  return r.ok;
}
```

### 5.8 `vox/list`

```vox
import "vox/list" as lst
import "vox/loader" as ld

fn list_smoke() -> String {
  let mut files: Vec[ld.SourceFile] = Vec();
  files.push(ld.SourceFile { path: "src/main.vox", text: "fn main() -> i32 { return 0; }" });
  let r: lst.BuildResult = lst.graph_from_files(files);
  if !r.ok { return r.err; }
  return lst.graph_to_json(r.graph);
}
```

说明：`graph_to_json` 的每个 module 节点除 `path/files/imports` 外，还包含模块内声明聚合的 `effects/effect_classes/resource_reads/resource_writes` 字段；图级别还输出：
- `resource_conflicts`（模块级 `rw`/`ww`）。
- `resource_orders`（模块级冲突在已知依赖边下的保守顺序建议，方向 `from -> to`）。
- `executor_lanes`（按 `effect_classes` 聚合的执行器 lane 建议：`class/executor/modules`）。
- `module_schedule_hints`（模块级保守调度提示：`module/class/executor/mode`，其中 `mode` 为 `parallel_ok` 或 `serial_guarded`）。
- `functions`（函数级能力快照：`id/module/kind/owner/name/effects/effect_classes/resource_reads/resource_writes`）。
- `function_resource_conflicts`（函数级 `rw`/`ww`）。

## 6. 当前落地状态

1. [x] Stable 层最小示例：见本章第 5 节。
2. [x] `vox/*` 库级回归测试：`src/vox/public_api_contract_test.vox`。
3. [x] `vox/token` 初版（显式 `Pos` + `Off` + `File` + `FileSet` + `Position`）。
4. [x] `vox/types` façade 初版（`Config + CheckResult + Info`，后端复用 `vox/typecheck`）。
5. [x] `vox/internal/*` 首批下沉：`vox/internal/text`，并在 `vox/manifest` 中复用。
6. [x] CLI 主流程复用 `vox/internal/text.trim_space`（`src/main.vox` 不再维护重复 trim helper）。
7. [x] `vox/internal/path` 下沉落地：统一 `base_name/dirname/join/slash-normalize/is_abs_path` 等路径辅助逻辑，`src/main.vox` 与 `vox/macroexpand` 复用同一实现。
8. [x] Stable/Experimental 模块统一头注释（稳定性级别 + 迁移策略）。
9. [x] `vox/list`（go list 对标）：输出完整包依赖图（模块、导入边、可选 JSON），并附带模块级 effect/resource 能力聚合信息。
10. [x] `vox/internal/text` 第二批复用：`has_prefix/has_suffix/contains_str/trim_space` 在 `main/compile/loader/typecheck/world/fmt/list/manifest` 统一复用，减少重复 helper 实现。
11. [x] `vox/internal/strset` 下沉：`insert_sorted/sort/push_unique_sorted` 在 `main` 与 `vox/list` 统一复用，减少字符串集合排序/去重重复实现。
12. [x] `vox/internal/text` 第三批复用：`main_lock/main_toolchain` 统一改用 `txt.index_byte/split_lines/unquote_double_trimmed/has_prefix`，移除重复文本解析 helper。
13. [x] `vox/internal/text` 第四批复用：`vox/typecheck/tc_struct_lit`、`vox/irgen/async_lower` 统一改用 `txt.contains_str/txt.has_prefix`，减少跨阶段重复字符串 helper。
14. [x] `vox/internal/text` 第五批复用：`vox/typecheck/world` 移除 `has_prefix/contains_str` 转发 helper，直接调用 `txt.has_prefix/txt.contains_str`，进一步减少重复实现。
15. [x] `vox/internal/text` 第六批复用：`vox/manifest` 移除 `has_prefix/has_suffix/contains_str` 转发 helper，直接调用 `txt.*`，保持解析逻辑不变并降低维护面。
16. [x] `vox/internal/text` 第七批复用：`vox/manifest` 继续移除 `trim/index/split` 转发 helper，直接调用 `txt.trim_space/txt.index_byte/txt.split_*`，进一步减少解析层中间封装。
