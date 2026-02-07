# 语法细节（草案）

> 本章记录 Vox 的关键字、优先级与若干重要语法决策。未覆盖的语法按 Rust-like 默认直觉处理，后续逐步补全。

## 关键字（草案）

声明：

`fn struct enum trait impl type const static`

控制流：

`if else match for while loop break continue return`

绑定与可变性：

`let mut move`

编译期：

`comptime`

模块：

`import pub as from where`

不安全：

`unsafe`

## 泛型语法

- 泛型（Stage0 最小子集）：
  - 泛型函数声明：`fn id[T](x: T) -> T { ... }`
  - 泛型调用（可显式给出类型实参）：`id[i32](1)`
  - 泛型调用（通常可省略类型实参，由实参/返回期望推导）：`id(1)`
- 其它（Stage0 暂不实现）：`impl[T] ...`、const 泛型等。

## 运算符优先级

沿用 Rust-like 的常见规则（后续可给出完整表）。

## `if` 表达式（Stage0 增补，已定）

除语句形式的 `if { ... } else { ... }` 外，Vox 允许在**表达式位置**使用 `if`：

```vox
let x: i32 = if cond { 1 } else { 2 };
return if ok { a } else { b };
```

语法（Stage0 最小子集）：

- 分支使用 `{ ... }`，其中只允许**单个表达式**（不是语句块）。
- `else` 对表达式形式是**必需的**。
- 允许 `else if ...` 链式形式（其 `else` 分支本身是 `if` 表达式）。

类型规则（Stage0）：

- `cond` 必须是 `bool`。
- `then` 与 `else` 分支的类型必须一致（或在 untyped int 约束下可被推导为一致）。

## Union 类型语法（已定，草案）

类型位置允许 `|` 组合 union 类型：

```vox
type Value = I32: i32 | Str: String;
```

说明：

- 推荐写 `Label: Type`，使构造与匹配有稳定名字：`.I32(1)`、`.Str("x")`。
- 对简单名义类型允许省略 label：`type X = Foo | Bar;`。

## 字符串字面量

- `"..."`：类型为 `&'static str`（可按 `&str` 使用）

## 范围类型语法（已定，草案）

类型位置允许 `@range(lo..=hi) T`：

```vox
type Tiny = @range(0..=3) i8;
```

范围类型在运行时的检查语义（已定）：

- 当发生到范围类型的显式转换时：
  - 编译期可确定越界：编译错误
  - 否则：运行时检查，失败 **panic**
- 若需要可恢复错误，请使用返回 `Option/Result` 的转换函数（例如 `Tiny::try_from(...)`）。

## 枚举构造子点前缀简写（已定）

当枚举类型可由上下文确定时，允许用 `.Variant` 代替 `Enum.Variant`：

```vox
let x: Option[i32] = .Some(1);
let y: Result[i32, Err] = .Ok(1);

match x {
  .Some(v) => v,
  .None => 0,
}
```

上下文不足以确定枚举类型时，必须写全路径：`Option.Some(1)`。

## 禁止的引用语法位置

为配合“临时借用”规则：

- 非 `&'static` 的 `&T` / `&mut T` 不允许出现在 struct 字段与返回类型中（详见 `docs/07-memory-management.md`）

## 成员访问（`.`）

Vox 统一使用 `.` 表示“成员访问”，并在不同上下文中解析为不同含义：

- **值成员**：`expr.field`（结构体字段访问）
- **模块成员**：`module.name`（通过 `import` 引入的模块/依赖包的命名空间成员）

解析规则（Stage0 先实现最小子集）：

- `a.b(...)`（调用上下文）
  - 若 `a`（或 `a.b.c` 的根）是当前作用域中的局部变量/参数：尝试解析为**值方法调用**。
    - Stage0 仅支持一小部分**内建类型的 intrinsic 方法**（见下）。
    - 其它类型的方法调用在 Stage0 报错（Stage1 再引入 trait/impl）。
  - 否则：`a` 必须是本文件 `import "..." [as alias]` 引入的命名空间别名；解析为该命名空间下的函数调用。
- `a.b`（表达式上下文）
  - 若 `a` 的类型是 `struct`：解析为字段读取。
  - 其它类型：报错（Stage0 先不支持动态/反射式成员访问）。

### 内建 intrinsic 方法（Stage0 最小子集）

Stage0 为了减少 Stage1（编译器代码）的样板，内建支持：

- `Vec[T]`：
  - `v.push(x) -> ()`（可变）
  - `v.len() -> i32`
  - `v.get(i: i32) -> T`
- `String`：
  - `s.len() -> i32`
  - `s.byte_at(i: i32) -> i32`
  - `s.slice(start: i32, end: i32) -> String`

对 receiver 的约束（Stage0）：

- 非变更方法（`len/get/byte_at/slice`）：receiver 可以是任意表达式（例如 `ctx.items.len()`）。
- 变更方法（`Vec.push`）：receiver 必须是 **place**（可写位置）。
  - Stage0 当前支持：局部变量 `v.push(x)`，以及可变局部 struct 的直接字段 `s.items.push(x)`。
  - 其它更复杂的 place（例如多级字段）后续再扩展。

## 导入语法（Stage0 最小子集）

模块/包导入：

```vox
import "dep"
import "dep" as d
```

命名导入（可导入 `pub` 的函数与名义类型）：

```vox
import { one, Point as P } from "dep"

fn main() -> i32 {
  let p: P = P { x: 1, y: 2 };
  return one() + p.x;
}
```

说明（Stage0）：

- `import { ... } from "path"` 中每个名字会在目标命名空间中解析为函数或名义类型（`struct/enum`）。
- 若同名同时存在函数与类型，则报错 `ambiguous imported name`（Stage0 先不混用命名空间）。
