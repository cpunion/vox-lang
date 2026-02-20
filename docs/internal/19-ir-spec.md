# IR 规范（v0，当前实现）

本章定义 Vox 的 IR v0，用于：

- 作为当前 C 后端与可执行文件生成的稳定接口
- 作为编译前端到后端之间的中间层约束

注意：IR v0 重点描述基础 lowering 子集；高级能力（comptime/宏/trait 高级分派等）会在上层阶段处理后再降入本 IR。
文中出现的 `历史引导版本` 属于历史术语，用于解释演进背景，不代表当前仓库结构。

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

- 函数名：当前允许包含模块/包限定名（例如 `mathlib::one`、`utils.io::read`）。
  - 规范化表示：以 `::` 分隔包/模块与成员名；模块路径内部可用 `.` 连接。
- 块名：同函数名
- 临时值：`%t<number>`（例如 `%t0`）
- 参数引用：`%p<number>`（例如 `%p0`）
- 槽位（可变局部变量）：`$v<number>`（例如 `$v3`）

## 3. 类型

IR v0 基础类型：

- `i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`（当前实现支持的整数类型）
- `f32/f64`
- `bool`
- `unit`
- `str`（当前 `String` 文本类型）
- `struct(<qualified_name>)`（名义结构体类型）
- `enum(<qualified_name>)`（名义枚举类型）

Stage2（编译器内部类型池）补充说明：

- `TyKind.Ref` 表达借用形状（`&T/&mut T/&'static T/&'static mut T`），并非擦除保留到 IR（函数签名、slot/temp/call 类型索引可见借用形状）。
- `Range` 仍按 v0 设计在 irgen 侧降到基类型并通过 `range_check` 表达约束。
- C 后端把 `Ref` 作为透明包装映射到底层 C 类型，并在比较/名义等值路径中按底层类型语义处理。

说明（当前实现策略）：

  - `enum` 在 C 后端降低为 tagged union：
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

### 5.2 整数运算

```
%t2 = add i64 %t0 %t1
%t3 = sub i64 %t0 %t1
%t4 = mul i64 %t0 %t1
%t5 = div i64 %t0 %t1
%t6 = mod i64 %t0 %t1
%t7 = bitand i64 %t0 %t1
%t8 = bitor i64 %t0 %t1
%t9 = bitxor i64 %t0 %t1
%t10 = shl i64 %t0 %t1
%t11 = shr i64 %t0 %t1
```

说明：

- `add/sub/mul`：wrapping（按位宽截断）。
- `div/mod`：除零必须 panic；有符号 `MIN / -1` 与 `MIN % -1` 必须 panic。
- `shl/shr`：移位位数越界必须 panic（`shift count out of range`）。

### 5.2.1 浮点运算（v0）

`f32/f64` 复用 `add/sub/mul/div` 与比较指令：

```
%t0 = const f64 1.5
%t1 = const f64 2.0
%t2 = add f64 %t0 %t1
%t3 = cmp_lt f64 %t0 %t1
```

约束：

- `%`、`bitand/bitor/bitxor/shl/shr` 不适用于浮点类型。

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

说明：

- `and/or/not` 是对 **已求值** 的 `bool` 值进行运算，不具备短路行为。
- 源码层的 `&&/||` 具备短路语义（见 `docs/internal/14-syntax-details.md`），在 lowering 时应使用 CFG（`condbr` + blocks）来实现。

### 5.4.1 数值转换（cast）

IR v0 定义“整数到整数”的显式转换：

```
%t0 = int_cast i32 i64 %t1
%t2 = int_cast_checked i64 i32 %t3
%t4 = float_cast f32 f64 %t5
```

约束（当前实现）：

- `int_cast`：
  - 仅允许 `int -> int`。
  - 语义为不检查的显式转换（可能截断）。
- `int_cast_checked`：
  - 目标类型必须是整数。
  - 源类型允许 `int` 或 `float`。
  - 语义为 checked cast（越界/非法值 panic）。
- `float_cast`：
  - 目标类型必须是浮点。
  - 源类型允许 `int` 或 `float`。
  - 语义为数值转换（由后端目标语言转换规则承载）。
- `int` 类型集合：`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`
- `float` 类型集合：`f32/f64`

### 5.4.2 范围检查（`@range`）

`@range(lo..=hi) T` 在 IR v0 中**不新增类型表示**：其运行时表示与底层标量 `T` 相同。

进入范围类型（例如 `x as Tiny`，其中 `type Tiny = @range(0..=3) i32`）时，lowering 需插入范围检查指令：

```
range_check i32 0 3 %t0
range_check i64 0 3 %t0
```

约束：

- `range_check` 没有结果值，只做检查。
- 若值不在 `[lo, hi]`（包含端点）内，必须 `panic("range check failed")`。
- `range_check` 的类型参数当前允许所有整数标量类型：`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`。
- `range_check` 还要求 `lo <= hi`（非法区间在 IR 校验阶段报错）。

### 5.4.3 IR 校验（当前实现）

在当前编译管线中，IR 进入后端前会执行 `verify`：

- 类型索引必须有效（函数参数/返回、结构体字段、枚举 payload、相关指令）
- cast 指令必须满足本节定义的类型组合约束
- `binop`/`cmp` 必须满足操作符与操作数类型组合约束（如有序比较仅允许 `int/float/string`，位运算仅允许整数）
- `range_check` 必须作用于整数类型且区间合法（`lo <= hi`）
- `struct_init` 的类型必须是 `struct`，且初始化字段名必须存在于对应声明中
- `enum_init` 的类型必须是 `enum`，variant 必须存在且 payload 个数必须与声明一致
- `enum_payload` 的 payload index 必须为非负
- CFG 终结指令分支目标必须存在

校验失败时，编译直接返回错误，不进入 codegen。

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

结构体值在 IR v0 中是“按值”类型（在当前 C 后端中对应 C struct）。

结构体字面量初始化：

```
%t0 = struct_init struct(Point) { x: 1, y: 2 }
```

字段读取：

```
%t1 = field_get i32 %t0 .x
```

字段写入（先支持对局部 slot 的字段写入）：

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

### 5.9 Vec（v0 内建容器）

`Vec[T]` 在 v0 中作为内建类型，降低到 C 运行时的 `vox_vec`（元素按值拷贝，v0 暂无 drop glue）。

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

### 5.10 字符串运行时指令（v0）

`String` 在 C 后端中当前降低为 `const char*`，并提供最小运行时指令集：

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
%t2 = u64_to_str 123
%t3 = isize_to_str 123
%t4 = usize_to_str 123
%t5 = f32_to_str 1.5
%t6 = f64_to_str 1.5
%t7 = bool_to_str true
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
