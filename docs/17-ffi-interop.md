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
- `ffi_import` / `ffi_export` 函数不允许 variadic 参数。

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
| `isize/usize` | `int64_t/uint64_t` | Stage2 当前固定为 64 位 |
| `f32/f64` | `float/double` | IEEE 浮点 |
| `String` | `const char*` | UTF-8 字节串，NUL 结尾，调用方负责约定生命周期 |

不在白名单的类型（如结构体、枚举、Vec、引用、range、泛型实例等）在类型检查阶段报错。

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
