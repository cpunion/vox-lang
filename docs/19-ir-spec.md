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

- 函数名：`[A-Za-z_][A-Za-z0-9_]*`
- 块名：同函数名
- 临时值：`%t<number>`（例如 `%t0`）
- 参数引用：`%p<number>`（例如 `%p0`）
- 槽位（可变局部变量）：`$v<number>`（例如 `$v3`）

## 3. 类型

IR v0 只定义 stage0 必需类型：

- `i32`、`i64`
- `bool`
- `unit`

（`String` 在 stage0 解释器里存在，但 IR 后端暂不支持。）

## 4. 程序结构

一个 IR 文件包含：

- 头：`ir v0`
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
