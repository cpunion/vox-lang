# FFI 与互操作（草案）

## C FFI

```vox
extern "C" {
  fn malloc(size: usize) -> *mut u8;
  fn free(ptr: *mut u8);
}
```

导出符号（属性语法待定，示意）：

```vox
#[export_name = "vox_add"]
#[no_mangle]
extern "C" fn add(a: i32, b: i32) -> i32 { a + b }
```

## 布局与映射

- `#[repr(C)]`：保证与 C 兼容布局
- 指针映射：`*const T` / `*mut T`
- `&T` / `&mut T` 仅作为临时参数传递到 C（调用方负责保证有效期）

## 字符串

建议提供：

- `CStr`/`CString`（与 Rust 类似）
- 字面量 `c"..."`（是否引入该语法待定）

## WASM/JS

保留 `#[wasm_import(...)]` / `#[wasm_export]` 一类属性（细节 deferred）。

