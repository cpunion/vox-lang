# FFI 与互操作

## 语法

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

## 约束

- `@ffi_import` 仅允许标注在顶层函数声明上，且该函数必须以 `;` 结尾（无函数体）。
- `@ffi_import("c", "symbol")` 需要 2 个参数。
- `@ffi_import("wasm", "module", "symbol")` 需要 3 个参数。
- 单个函数最多一个 `@ffi_import`。
- `@ffi_export("c", "symbol")` 与 `@ffi_export("wasm", "symbol")` 需要 2 个参数。
- 单个函数可有多个 `@ffi_export`（不同 target）。
- 同一函数上禁止同时出现 `@ffi_import` 与 `@ffi_export`。
- `@ffi_export` 函数必须是 `pub fn` 且必须有函数体。
- v0 限制：FFI 函数不支持类型参数/常量参数与可变参数。

## v0 FFI 类型白名单

- `()`
- `bool`
- `i8 i16 i32 i64 isize`
- `u8 u16 u32 u64 usize`
- `f32 f64`
- `String`（当前 C 后端按 `const char*` 处理）

超出白名单的类型（结构体、枚举、Vec、引用、range、泛型实例等）在类型检查阶段报错。

## C 后端行为（v0）

- `@ffi_import("c", "...")` 生成 `extern` 函数声明，并在调用点直连外部符号。
- `@ffi_export("c", "...")` 生成导出包装函数，包装内部 mangled Vox 函数名。
- `@ffi_import("wasm", "module", "symbol")` 在 C 后端会生成 wasm import 属性（clang 风格 `import_module/import_name`）。
- `@ffi_export("wasm", "symbol")` 在 C 后端会生成 wasm export 属性（clang 风格 `export_name`）。

## 构建导出库

- 可执行（默认）：`vox build-pkg out.bin`
- 共享库：`vox build-pkg --artifact=shared libvox.so`（macOS 常用 `.dylib`，Windows 常用 `.dll`）
- 静态库：`vox build-pkg --artifact=static libvox.a`（Windows MSVC 常用 `.lib`）

`shared/static` 产物不会生成 driver `main`。

## WASM 状态

- `wasm` 导入/导出属性已可在 C 后端生成对应属性声明（实验态）。
- 当前仍是“C 后端 + 外部 wasm 工具链”路线，不是独立 wasm IR/后端。
- 构建示例：`vox build-pkg --target=wasm32-unknown-unknown out.wasm`
