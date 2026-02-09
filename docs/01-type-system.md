# 类型系统

## 目标

1. 静态强类型：所有类型错误在编译期发现。
2. 类型推断：减少样板标注，保留可预测性。
3. 泛型与 trait：支持零成本抽象与可复用库。
4. ADT：`struct`/`enum` + `match`。

## 基础类型

数值类型（草案）：

- 有符号：`i8 i16 i32 i64 isize`
- 无符号：`u8 u16 u32 u64 usize`
- 浮点：`f32 f64`
- 其它：`bool char`

Stage0/Stage1 v0 当前实现状态（2026-02）：

- `f32/f64` 已支持基础表达式语义：字面量、`+ - * /`、比较（`< <= > >=`）与相等（`== !=`）。
- `const` 已支持 `f32/f64` 字面量、常量引用、`-x`、`== !=` 与 `f32 <-> f64 as`（浮点算术/比较折叠后续补齐）。
- `@range` 仍仅支持整数底层类型（不支持 `f32/f64` 作为 range base）。

字符串：

- `str`：不定长，只能通过 `&str` / `&'static str` 使用
- `String`：拥有的字符串

复合类型：

- 数组：`[T; N]`
- 切片：`[T]`（只能通过 `&[T]` 使用）
- 元组：`(T, U, ...)`

## 结构体

```vox
struct Point {
  x: f32,
  y: f32,
}

let p = Point { x: 1.0, y: 2.0 }
```

字段访问与更新（草案，Stage0 优先支持）：

```vox
let mut p = Point { x: 1, y: 2 };
let a = p.x;
p.x = a + 1;
```

## 枚举（代数数据类型）

```vox
enum Option[T] {
  Some(T),
  None,
}

enum Result[T, E] {
  Ok(T),
  Err(E),
}
```

## 模式匹配

```vox
fn show(opt: Option[i32]) -> i32 {
  match opt {
    Option.Some(v) => v,
    Option.None => 0,
  }
}
```

Vox 支持枚举构造子的“点前缀简写”（当枚举类型可由上下文确定时）：

```vox
let opt: Option[i32] = .Some(42);
let opt: Option[i32] = .None;
```

无法从上下文确定枚举类型时，必须使用全路径（例如 `Option.Some(42)`）。

## Union 类型（`|`，已定）

Vox 支持 union 类型（sum type）表达式，并允许 union 的每一支为**任意类型**：

```vox
type Value = I32: i32 | Str: String | Bytes: Vec[u8];
```

说明（草案）：

- `A | B` 是一种类型表达式。
- 当 union 的分支需要稳定名字时，使用 `Label: Type` 形式指定构造子/模式的名字。
- 若分支类型是简单的名义类型（例如 `Foo`），允许省略 label，并默认 label 为 `Foo`：

```vox
type X = Foo | Bar; // 等价于: Foo: Foo | Bar: Bar
```

构造与匹配示例：

```vox
fn demo(v: Value) -> i32 {
  match v {
    .I32(x) => x,
    .Str(_) => 0,
    .Bytes(b) => b.len() as i32,
  }
}

let v: Value = .I32(42);
```

表示与性能（草案）：

- union 默认使用“tagged”表示（等价于一个匿名 `enum`）。
- 编译器可在满足条件时做布局压缩（niche/spare bits）与分支消除，但不改变语义。

## 范围类型（`@range`，已定）

Vox 支持对标量类型附加“值域约束”，用于表达与优化。

```vox
type Tiny = @range(0..=3) i32;
```

规则（已定，草案表述）：

- `@range(lo..=hi) T` 只能用于离散标量类型（细分范围待定）。
- 范围类型的运行时表示与底层标量 `T` 相同（零额外存储）。
- **检查发生在编译期与类型转换边界**（“进入范围类型”时检查）：
  - 若被转换值在编译期可确定：越界为编译错误。
  - 否则在转换点插入检查；失败时 **panic**。
- 在普通算术/赋值中不强制维持不变量；仅在“进入范围类型”时检查。

Stage0/Stage1 v0 当前实现限制：

- `T` 仅支持整数类型（当前 stage0/stage1 实现：`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`）。
- `lo/hi` 仅支持十进制整数字面量（更多标量与字面量形式后续再扩展）。

建议提供的转换 API（草案）：

```vox
impl Tiny {
  // 失败时 panic（用于显式窄化/类型转换）
  fn from(x: i32) -> Tiny;

  // 显式可恢复：返回 Option/Result
  fn try_from(x: i32) -> Option[Tiny];
  fn try_from_result(x: i32) -> Result[Tiny, RangeError];

  unsafe fn from_unchecked(x: i32) -> Tiny;
}
```

约定（草案）：`x as Tiny` 等价于 `Tiny::from(x)`（带边界检查，失败 panic）。

编译器也可在可证明安全时消除检查，并允许将 `try_from/try_from_result` 优化为无检查构造：

```vox
fn f(x: i8) -> Option[Tiny] {
  if x < 0 || x > 3 { return .None; }
  // 这里可证明安全：Tiny::try_from(x) 可被优化为无检查构造
  Tiny::try_from(x)
}
```

## 泛型

Vox 的泛型使用方括号：

```vox
fn identity[T](x: T) -> T { x }

struct Pair[T, U] { first: T, second: U }
```

Const 泛型（草案，细节见 `docs/06-advanced-generics.md`）：

```vox
struct Array[T, const N: usize] {
  data: [T; N],
}
```

## Trait

```vox
trait Display {
  fn display(&self) -> String;
}

impl Display for Point {
  fn display(&self) -> String { format("({}, {})", self.x, self.y) }
}
```

说明（草案）：

- trait bound：`T: Trait1 + Trait2`
- `where` 子句支持复杂约束
- 关联类型、GAT、HRTB 等高级能力暂缓引入，优先保证可实现性与诊断质量

## 类型推断（草案）

- 局部变量可推断：`let x = 42`
- 泛型实参优先从参数推断，必要时支持 turbofish：`f::[i32](1)`
