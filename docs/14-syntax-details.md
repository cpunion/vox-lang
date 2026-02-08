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

## 相等（`==`/`!=`，Stage0 约束）

Stage0 为了保持实现范围可控，对相等运算符有额外约束：

- `bool/i32/i64/String`：支持完整 `==`/`!=`。
- `enum`：仅支持与 **unit variant**（无 payload 的构造子值）比较，例如 `x == E.None`。
  - 该比较降低为 `enum_tag(x) == tag(E.None)`。
  - 不支持 `E.A(1) == E.A(2)` 这类 payload 比较（Stage1 再引入更完整的机制）。

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

语法：

- `"..."`：字符串字面量

转义（Stage0/Stage1 当前保证的最小集合）：

- `\\`：反斜杠
- `\"`：双引号
- `\n`：换行（LF）
- `\r`：回车（CR）
- `\t`：制表符（TAB）

说明：

- 解析阶段会把转义序列还原成真实字符（例如源码 `\"a\\n\"` 的值包含单个换行字符）。
- 后端生成 C 字符串字面量时会再次做 C 级别转义（例如把换行字符输出为 `\\n`），以保证生成的 `.c` 文件语法正确。

类型（设计草案 vs 当前实现）：

- 语言设计草案倾向将 `"..."` 视为 `&'static str`（可按 `&str` 使用）。
- 但 Stage0/Stage1 的自举子集里暂时将字符串字面量视为 `String`（后端以 `const char*` 表示），以减少引入借用类型系统的实现负担；未来引入 `&str` 后再统一语义。

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
  - `v.join(sep: String) -> String`（仅当 `T == String`；用于拼接字符串片段）
- `String`：
  - `s.len() -> i32`
  - `s.byte_at(i: i32) -> i32`
  - `s.slice(start: i32, end: i32) -> String`
  - `s.concat(other: String) -> String`（返回新字符串；用于代码生成等场景）
  - `s.escape_c() -> String`（返回可放入 C 字符串字面量的转义内容）
- `i32/i64/bool`：
  - `x.to_string() -> String`（最小格式化能力；用于诊断与代码生成）

对 receiver 的约束（Stage0）：

- 非变更方法（`len/get/byte_at/slice`）：receiver 可以是任意表达式（例如 `ctx.items.len()`）。
- 变更方法（`Vec.push`）：receiver 必须是 **place**（可写位置）。
  - Stage0 当前支持：局部变量 `v.push(x)`，以及可变局部 struct 的直接字段 `s.items.push(x)`。
  - 其它更复杂的 place（例如多级字段）后续再扩展。

### 保留的 `__*` 低层 intrinsic（Stage0/Stage1 自举期）

除 `panic/print` 外，自举期还存在少量以 `__` 开头的低层 intrinsic（用于 `std/fs`、`std/process` 等最小工具链能力）。

约束（Stage0/Stage1）：

- **非 `std/**` 模块禁止直接调用**以 `__` 开头的函数（例如 `__exec(...)`）。
- 禁止用户代码定义以 `__` 开头的函数/类型名（保留给自举期 intrinsic 与标准库实现）。
- 这些名字保留给标准库实现与自举工具链，用户代码应通过 `std/fs`、`std/process` 等封装接口使用。

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
