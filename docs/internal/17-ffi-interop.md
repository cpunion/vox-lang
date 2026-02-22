# FFI 与互操作

本章定义 Stage2 当前生效的 FFI 属性模型、ABI 映射和后端生成规则。

## 1. 属性语法

导入外部符号：

```vox
@ffi_import("c", "puts")
fn c_puts(s: String) -> i32;

@ffi_import("wasm", "env", "log_i32")
fn wasm_log(v: i32) -> ();
```

导出 Vox 函数：

```vox
@ffi_export("c", "vox_add")
@ffi_export("wasm", "vox_add")
pub fn add(a: i32, b: i32) -> i32 {
  return a + b;
}
```

参数签名：

- `@ffi_import("c", symbol)`
- `@ffi_import("wasm", module, symbol)`
- `@ffi_export("c", symbol)`
- `@ffi_export("wasm", symbol)`

## 2. 属性约束

### 2.1 位置与组合约束

- `@ffi_import` 只能标注在顶层函数声明，且函数必须无 body（以 `;` 结束）。
- 单个函数最多一个 `@ffi_import`。
- `@ffi_export` 只能标注在顶层函数定义（必须有 body）。
- `@ffi_export` 函数必须是 `pub fn`。
- 同一函数禁止同时出现 `@ffi_import` 与 `@ffi_export`。
- 单个函数可有多个 `@ffi_export`，但 target 不可重复（同一函数上不能两个 `@ffi_export("c", ...)`）。

### 2.2 泛型与参数形态约束

- `ffi_import` / `ffi_export` 函数不允许类型参数。
- `ffi_import` / `ffi_export` 函数不允许常量泛型参数。
- `ffi_import` / `ffi_export` 函数当前不支持 variadic 参数（会报错拒绝）。

### 2.3 符号唯一性约束

- `ffi_export` 在包级做冲突检查：同一 `target + symbol` 组合只能出现一次。
- 不同 target 可复用相同 symbol（例如同时导出给 C 和 wasm）。

## 3. ABI 映射（Stage2 v0）

当前走 C 后端，因此 FFI ABI 以 C 形态为准，再由目标工具链映射到目标平台（含 wasm）。

| Vox 类型 | C 侧类型 | 说明 |
|---|---|---|
| `()` | `void` | 无返回值 |
| `bool` | `bool` | C `_Bool` / `<stdbool.h>` |
| `i8/i16/i32/i64` | `int8_t/int16_t/int32_t/int64_t` | 定宽有符号整型 |
| `u8/u16/u32/u64` | `uint8_t/uint16_t/uint32_t/uint64_t` | 定宽无符号整型 |
| `isize/usize` | `intptr_t/uintptr_t` | 按目标指针宽度映射（32/64） |
| `rawptr` | `void*` | FFI 互操作专用的不透明指针类型 |
| `f32/f64` | `float/double` | IEEE 浮点 |
| `String` | `const char*` | 兼容映射（历史保留）；新接口不建议把文本边界直接建模为 C-string |
| `Vec[String]` | `vox_vec` | 运行时 `Vec` 句柄（元素为 `const char*`） |

不在白名单的类型（如结构体、枚举、除 `Vec[String]` 外的 `Vec`、`&T`、range、泛型实例等）在类型检查阶段报错。

补充：FFI 参数位置允许有限的 inout 形态 `&mut Scalar`（例如 `&mut i32`），用于对接 C 的 `T*` 输出参数；返回值仍限制为普通 ABI 值，不允许返回引用。

### 3.1 文本/字节边界约定（A44）

当前约定：

- 新增跨边界 I/O 接口优先使用 `rawptr + len`（例如 `write(fd, p, n)`）。
- `String` 形参仅用于必须传递 C 风格路径/命令等接口，作为兼容映射保留。
- 需要 NUL 终止时，由适配层显式构造终止缓冲；业务 API 不应隐式依赖“输入文本天然带 `\\0`”。

### 3.2 当前 `String -> C` 边界盘点（A44-1）

为推进 `ptr + len` 收敛，当前按“可先迁移”和“暂时保留”分层：

- 优先迁移（字节载荷）：
  - `std/io::NetConn.try_send`：已落地为 `std/sys::send(handle, ptr, len)`（平台 `@build` 分支直接绑定 OS `send`），不新增 `vox_host_*` 网络桥接符号。
  - 其它新增跨边界 payload API：默认禁止直接使用 `String` 作为 C ABI 文本载荷。
- 暂时保留（路径/命令/环境）：
  - `std/sys::{open/access/mkdir/creat/system}` 及其平台分支。
  - `std/runtime::{read_file/write_file/path_exists/mkdir_p/exec/walk_files/getenv}`。

说明：

- 以上“暂时保留”并不代表最终形态；仅用于在当前 bootstrap 约束下维持可发布链路。
- 完成 A44-1 时，应逐项把“可先迁移”的载荷型接口切换到 `ptr + len`，并补齐回归测试。

## 4. 代码生成模型（C 后端）

### 4.1 `@ffi_import("c", symbol)`

- 生成 `extern` 声明。
- 调用点直接调用 `symbol`。

### 4.2 `@ffi_export("c", symbol)`

- 生成导出包装函数 `symbol(...)`。
- 包装函数内部调用 mangled 的 Vox 函数实现。

### 4.3 `@ffi_import("wasm", module, symbol)`

- 生成 `import_module/import_name` 属性（clang 风格）。
- C 标识符使用内部别名（`vox_imp_*`），避免和用户符号冲突。
- 真实导入名由 `import_name(symbol)` 决定。

### 4.4 `@ffi_export("wasm", symbol)`

- 生成 `export_name` 属性（clang 风格）。
- 包装函数 C 标识符使用内部别名（`vox_exp_*`），避免重定义冲突。
- 真实导出名由 `export_name(symbol)` 决定。

## 5. 构建与目标组合

- `vox build` 支持 `--artifact=exe|static|shared`。
- `shared/static` 产物不生成 driver `main`。
- wasm 目标当前仅支持 `--artifact=exe`。

wasm 三元组：

- `wasm32-unknown-unknown`
- `wasm32-wasi`
- `wasm32-unknown-emscripten`

测试运行约束：

- `test` 在 wasm 上仅支持 `wasm32-wasi`。
- runner 由 `WASM_RUNNER` 指定，默认 `wasmtime`。

## 6. 运行时/互操作说明

- 当前 wasm 路线是 “C 后端 + 外部 wasm 工具链”，不是独立 wasm IR/后端。
- `examples/wasm_call` 中 Node/Web 示例为了直接调用导出函数，提供了最小 `wasi_snapshot_preview1` stub。
- 该 stub 仅用于 smoke/纯计算调用演示，不等同完整 WASI 运行时。

## 7. 示例

- Node.js + 浏览器调用 wasm：`examples/wasm_call/`
  - 构建：`vox build --target=wasm32-unknown-unknown target/vox_wasm_demo.wasm`
  - Node 运行：`node node/run.mjs`
  - Web 运行：`VOX_WASM_WEB_PORT=8080 node web/server.mjs` 后访问 `http://127.0.0.1:8080/web/`
