# IR 规范（v0，Stage0）

本章定义 Vox 的 IR v0（stage0 使用），用于：

- 作为 stage0 后端与可执行文件生成的稳定接口
- 作为 stage1 自举编译器可对齐的中间层

注意：IR v0 只覆盖 stage0 子集，不包含 comptime/宏/trait/泛型等。

## 1. 文件格式

IR v0 是文本格式，UTF-8，按行组织。

```
ir v0
fn main() -> i32
block entry:
  %t0 = const i32 0
  ret %t0
```

## 2. 标识符

- 函数名：stage0 当前允许包含模块/包限定名（例如 `mathlib::one`、`utils.io::read`）。
  - 规范化表示：以 `::` 分隔包/模块与成员名；模块路径内部可用 `.` 连接。
- 块名：同函数名
- 临时值：`%t<number>`（例如 `%t0`）
- 参数引用：`%p<number>`（例如 `%p0`）
- 槽位（可变局部变量）：`$v<number>`（例如 `$v3`）

## 3. 类型

IR v0 只定义 stage0 必需类型：

- `i32`、`i64`
- `bool`
- `unit`
- `str`（当前用于 stage0 的 `String` 字面量与最小字符串比较/返回；后续会替换为真正的字符串/切片模型）
- `struct(<qualified_name>)`（名义结构体类型）
- `enum(<qualified_name>)`（名义枚举类型）

说明（Stage0 实现策略）：

- `enum` 在 stage0 C 后端降低为 tagged union：
  - `tag: i32`（variant index）
  - `union { ... } payload`（每个 variant 一个 union member；每个 member 是一个 `struct`，字段名为 `_%d`；payload arity 支持 0..N）

## 4. 程序结构

一个 IR 文件包含：

- 头：`ir v0`
- 若干结构体定义（可选）：`struct <name> { <fields...> }`
- 若干枚举定义（可选）：`enum <name> { <variants...> }`
- 若干函数：`fn <name>(<params>) -> <ret>`
- 函数包含若干 block（CFG）

## 5. 指令集（最小子集）

### 5.1 常量

```
%t0 = const i64 123
%t1 = const bool true
```

### 5.2 算术

```
%t2 = add i64 %t0 %t1
%t3 = sub i64 %t0 %t1
%t4 = mul i64 %t0 %t1
%t5 = div i64 %t0 %t1
%t6 = mod i64 %t0 %t1
```

### 5.3 比较

比较结果为 `bool`：

```
%t7 = cmp_lt i64 %t0 %t1
%t8 = cmp_eq i64 %t0 %t1
```

### 5.4 逻辑

```
%t9  = and %t7 %t8
%t10 = or  %t7 %t8
%t11 = not %t7
```

### 5.5 局部槽位（可变变量）

槽位用于降低 `let mut` 与赋值：

```
$v0 = slot i64
store $v0 %t0
%t1 = load i64 $v0
```

### 5.6 调用

```
%t0 = call i32 foo(%t1, %t2)
call unit bar(%t0)
```

### 5.7 结构体（struct）

结构体值在 IR v0 中是“按值”类型（在 stage0 C 后端中对应 C struct）。

结构体字面量初始化：

```
%t0 = struct_init struct(Point) { x: 1, y: 2 }
```

字段读取：

```
%t1 = field_get i32 %t0 .x
```

字段写入（Stage0 先支持对局部 slot 的字段写入）：

```
store_field $v0 .x %t1
```

### 5.8 枚举（enum）

构造（variant index 由 `enum` 定义顺序决定）：

```
%t0 = enum_init enum(Option) Some(1)
%t1 = enum_init enum(Option) None
%t2 = enum_init enum(Pair) Pair(1, 2)
```

读取 tag：

```
%t2 = enum_tag %t0
```

读取 payload 字段（仅当该 variant 带 payload，`index` 为 0-based）：

```
%t3 = enum_payload i32 %t0 Some 0
%t4 = enum_payload i32 %t2 Pair 1
```

### 5.9 Vec（Stage0 内建容器）

Stage0 将 `Vec[T]` 作为内建类型，降低到 C 运行时的 `vox_vec`（元素按值拷贝，v0 暂无 drop glue）。

构造新 vec：

```
%t0 = vec_new vec(i32)
```

push（receiver 必须是 slot）：

```
vec_push $v0 1
```

len/get：

```
%t1 = vec_len $v0
%t2 = vec_get i32 $v0 0
```

当 `T == str` 时，支持 join：

```
%t3 = vec_str_join $v1 ","
```

### 5.10 字符串运行时指令（Stage0）

Stage0 的 `String` 在 C 后端中当前降低为 `const char*`，并提供最小运行时指令集：

```
%t0 = str_len %t1
%t2 = str_byte_at %t1 0
%t3 = str_slice %t1 1 3
%t4 = str_concat %t1 %t3
%t5 = str_escape_c %t1
```

数值到字符串（最小格式化能力）：

```
%t0 = i32_to_str 123
%t1 = i64_to_str 123
%t2 = bool_to_str true
```

## 6. 终结指令（terminator）

每个 block 必须以 terminator 结束：

```
ret
ret %t0
br next
condbr %t0 then_blk else_blk
```

## 7. 校验规则（最小）

- 所有使用到的 `%tN` 必须在同一函数中先定义后使用
- `load/store` 的类型必须与 slot 声明一致
- CFG 中跳转目标必须存在
- `ret` 的值类型必须与函数返回类型一致（`unit` 返回值可省略）
