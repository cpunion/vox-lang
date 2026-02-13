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
  - 泛型参数 trait 约束：`fn eq[T: Eq + Show](x: T) -> bool { ... }`
  - `where` 约束：`fn eq[T](x: T) -> bool where T: Eq + Show { ... }`
  - `where comptime` 约束（Stage1 已实现子集）：`fn addn[const N: i32](x: i32) -> i32 where comptime N > 0, comptime N <= 8 { ... }`
  - `where comptime` 右值可为常量参数：`fn f[const N: i32, const M: i32](x: i32) -> i32 where comptime N < M { ... }`
  - `where comptime` 右值可为反射项：`fn f[T, U](x: T, y: U) -> i32 where comptime @size_of(T) <= @align_of(U) { ... }`
  - `where comptime` 支持类型布局反射约束：`fn fit[T](x: T) -> i32 where comptime @size_of(T) <= 64, comptime @align_of(T) <= 8 { ... }`
  - `where comptime` 支持字段/variant 数反射约束：`fn fit[T](x: T) -> i32 where comptime @field_count(T) <= 8 { ... }`
  - `where comptime` 支持类型 ID 反射约束：`fn fit[T](x: T) -> i32 where comptime @type(T) > 0 { ... }`
  - generic struct/enum 也支持 `where comptime`：`struct Small[T] where comptime @size_of(T) <= 8 { v: T }`
  - 泛型调用（可显式给出类型实参）：`id[i32](1)`
  - 泛型调用（通常可省略类型实参，由实参/返回期望推导）：`id(1)`
  - 函数 const 泛型：`fn addn[const N: i32](x: i32) -> i32 { return x + N; }`
  - const 泛型调用（显式 const 实参）：`addn[3](4)`
  - pack/variadic 语法（Stage2 当前实现）：`fn zip[T...](xs: T...) -> i32 { ... }`
    - `xs: T...` 会在类型检查阶段降级为 `Vec[T]`，并可进入 IR/codegen。
    - `T...`（type param pack 声明）当前只保留声明语义（最多 1 个且必须在末尾），未做真正 pack 展开。
- 其它（Stage0 暂不实现）：`impl[T] ...` 等。

## Trait 语法（Stage1 v0）

- trait 方法声明（无默认实现）：`trait Eq { fn eq(a: Self, b: Self) -> bool; }`
- trait 方法可带泛型参数与约束：`trait Wrap { fn wrap[T: Eq](x: Self, v: T) -> T where T: Show; }`
- trait 方法可带 const 泛型参数：`trait AddN { fn addn[const N: i32](x: Self, v: i32) -> i32; }`
- trait 方法默认实现：`trait Show { fn show(x: Self) -> String { return "x"; } }`
- 关联类型声明：`trait Iter { type Item; fn next(x: Self) -> Self.Item; }`
- supertrait：`trait Child: Parent + Other { ... }`
- `impl` 可省略带默认实现的方法（同模块/跨模块 trait 均可继承默认实现）
- `impl` 方法的泛型参数名按位置匹配 trait 方法（名称可不同）
- `impl` 方法的 const 泛型参数当前要求与 trait 方法同名同类型（按位置匹配）
- `impl` 需为 trait 中每个关联类型给出绑定：`impl Iter for I { type Item = i32; ... }`
- 支持在类型位置引用 `Self.Assoc`（trait/impl 方法签名）以及 `T.Assoc`（泛型签名，`T` 为类型参数）。
- `T.Assoc` 约束规则：`T` 的 trait bounds 中必须且只能有一个 trait 声明该关联类型，否则报错（unknown/ambiguous projection）。

## 运算符优先级

当前已实现子集的优先级（从高到低）：

1. postfix：
   - 成员访问：`a.b`
   - 调用：`f(x)` / `a.b(x)`
   - 调用的显式类型实参：`f[T](x)`（`[T]` 绑定到紧随其后的 `(...)`）
   - 显式转换（cast）：`expr as Type`
2. unary：`+x`、`-x`、`!x`
3. 乘法：`* / %`
4. 加法：`+ -`
5. 移位：`<< >>`
6. 按位与：`&`
7. 按位异或：`^`
8. 按位或：`|`
9. 比较：`< <= > >=`
10. 相等：`== !=`
11. 逻辑与：`&&`
12. 逻辑或：`||`

结合性（当前约定）：

- 所有二元运算符均为左结合，例如 `1 - 2 - 3` 解析为 `(1 - 2) - 3`。

## 赋值与复合赋值（Stage0/Stage1 v0）

当前赋值是**语句**，不是表达式：

- `x = e;`
- `x += e; x -= e; x *= e; x /= e; x %= e;`
- `x &= e; x |= e; x ^= e; x <<= e; x >>= e;`
- 字段同样支持：`s.f = e; s.f += e; ...`
- 多级字段也支持：`a.b.c = e; a.b.c += e; ...`

当前实现采用 parser 反糖（desugar）：

- `x += y;` 等价于 `x = x + y;`
- `s.f <<= y;` 等价于 `s.f = s.f << y;`

这保证 typecheck/codegen 仍沿用已有 `Assign` + `Binary` 语义路径。

## 逻辑运算（`&&` / `||`）

求值顺序与短路（Stage0/Stage1 v0 已定）：

- 求值顺序：从左到右。
- `a && b`：先求值 `a`；若 `a == false`，则 `b` **不会**被求值，结果为 `false`。
- `a || b`：先求值 `a`；若 `a == true`，则 `b` **不会**被求值，结果为 `true`。

## 相等（`==`/`!=`，Stage1 约束）

Stage1 当前对相等运算符的约束：

- `bool/<int>/f32/f64/String`：支持完整 `==`/`!=`。
  - 其中 `<int>` 指整数标量类型：`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`。
- `struct`：支持同类型结构体的 `==`/`!=`，逐字段比较。
  - 字段类型也必须是可比较类型（基础标量、`String`、或同样满足条件的 `struct/enum`）。
- `enum`：支持同类型枚举的 `==`/`!=`，包含 payload 比较（先比 tag，再按 variant 逐字段比较）。
  - payload 字段类型也必须是可比较类型（基础标量、`String`、或同样满足条件的 `struct/enum`）。

## 有序比较（`< <= > >=`，Stage1 约束）

- `bool`：不支持有序比较。
- `<int>`、`f32/f64`：支持同类型比较。
- `String`：支持字典序比较（按字节序，等价 `strcmp` 语义）。
- 泛型参数：当 `T` 显式带有 `Ord` bound（`T: Ord`）时，允许在函数体中使用 `< <= > >=` 比较。

## 浮点字面量（Stage0/Stage1 v0）

当前已实现的最小语法：

- `DIGITS "." DIGITS`（例如 `0.0`、`3.14`、`42.5`）
- 指数形式：`DIGITS [ "." DIGITS ] [e|E] [+-]? DIGITS`（例如 `1e3`、`2.5e-2`）
- 显式后缀：`f32` / `f64`（例如 `1.0f32`、`3e2f64`）

类型推导（当前实现）：

- 有显式后缀时：字面量类型固定为后缀类型；与期望类型冲突时报错。
- 有期望类型且为 `f32/f64` 时，按期望类型约束。
- 无期望类型时，默认推导为 `f64`。

## 整数运算（Stage0/Stage1 v0，已定）

对整数标量类型（`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`），当前语义约束为：

- 整数字面量（Stage0/Stage1）：
  - 当上下文期望 `u64/usize`（如显式类型注解、const 声明类型）时，十进制字面量允许完整 `0..18446744073709551615`。
  - 显式 cast 场景同样适用（例如 `18446744073709551615 as u64`）。
  - 无期望类型的字面量仍按 `i64` 处理（超出 `i64` 范围会报错）。
- `+ - *`：wrapping（按位宽截断），不 panic。
- `& | ^`：按位运算（按位宽解释），不 panic。
- `<< >>`：
  - 移位位数必须在 `[0, bit_width(T)-1]`，否则 `panic("shift count out of range")`。
  - `<<`：按位左移，结果按位宽截断。
  - `>>`：有符号整数为算术右移；无符号整数为逻辑右移。
- `/ %`：
  - 除以 0 必须 `panic("division by zero")`。
  - 对有符号类型（`i8/i16/i32/i64/isize`），`MIN / -1` 与 `MIN % -1` 必须 `panic("division overflow")`（避免后端 UB）。
- `expr as <int>`：整数到整数的显式转换是 **checked cast**：
  - 若编译期可确定溢出：编译错误。
  - 否则在转换点插入运行时检查；越界必须 panic（错误消息由后端决定）。

## 浮点运算（Stage0/Stage1 v0）

对 `f32/f64`，当前语义约束为：

- `+ - * /`：仅允许两侧同类型（`f32` 对 `f32`，`f64` 对 `f64`）。
- `< <= > >=`、`== !=`：同样要求两侧同类型的浮点。
- `%`：支持浮点（`f32/f64`，与 C 的 `fmodf/fmod` 语义一致）。
- `<< >>`：不支持浮点（仅整数）。
- `& | ^`：支持整数与 `bool`。
- 显式转换 `as` 支持：
  - `f32 <-> f64`
  - `int -> float`（允许，按目标浮点类型转换）
  - `float -> int`（checked cast：非有限值或越界会 panic）

## `if` 表达式（Stage0 增补，已定）

除语句形式的 `if { ... } else { ... }` 外，Vox 允许在**表达式位置**使用 `if`：

```vox
let x: i32 = if cond { 1 } else { 2 };
return if ok { a } else { b };
```

语法（Stage0 最小子集）：

- 分支使用 **表达式块**：`{ stmt*; tailExpr? }`。
  - `tailExpr` 可省略：省略时该分支值为 `()`。
- `else` 对表达式形式是**必需的**。
- 允许 `else if ...` 链式形式（其 `else` 分支本身是 `if` 表达式）。

类型规则（Stage0）：

- `cond` 必须是 `bool`。
- `then` 与 `else` 分支的类型必须一致（或在 untyped int 约束下可被推导为一致）。

## 错误传播语法（Stage1 v0）

- 后缀 `?`：`expr?`
  - 当前实现支持：
    - `Result[T, E]`：`Ok(v)` 继续，`Err(e)` 提前返回 `Err(e)`。
    - `Option[T]`：`Some(v)` 继续，`None` 提前返回 `None`。
  - 约束：
    - `?` 仅可用于 `Result/Option`。
    - 所在函数返回类型需与传播容器同类（`Result` 或 `Option`）。
    - `Result` 的 `Err` 类型规则：
      - 若源 `Err` 可直接赋给目标 `Err`，则直接传播；
      - 否则尝试 `std/prelude::Into`（源 `Err` 的 `Into::Target` 需兼容目标 `Err`，当前要求 `into` 为非泛型方法）。
- `try { ... }`
  - 当前实现为块级传播边界：
    - 块内 `?` 失败仅提前结束 `try` 块并返回 `Err/None`，不会直接退出外层函数。
    - 块正常完成时，尾表达式会自动包装为 `Ok/Some`。
    - 若尾表达式已是目标容器类型（`Result/Option`），则直接透传。

## Union 类型语法（已定，草案）

类型位置允许 `|` 组合 union 类型：

```vox
type Value = I32: i32 | Str: String;
```

说明：

- 推荐写 `Label: Type`，使构造与匹配有稳定名字：`.I32(1)`、`.Str("x")`。
- 对简单名义类型允许省略 label：`type X = Foo | Bar;`。

当前实现（Stage0/Stage1 v0）：

- 仅支持 **labeled** 形式：`type Name = A: TA | B: TB | ...`（每个 variant 当前只有 1 个 payload 类型）。
- 语义上等价于同名 `enum` 声明（即“tagged union”），并复用 `enum` 的构造与 `match` 机制。
  - 例如 `type Value = I32: i32 | Str: String;` 等价于 `enum Value { I32(i32), Str(String) }`。
  - 因此 `Value.I32(1)`、以及在期望类型已知时的简写 `.I32(1)` 都可用。

## 字符串字面量

语法：

- `"..."`：字符串字面量
- `"""..."""`：多行字符串字面量（支持自动 unindent）

转义（Stage0/Stage1 当前保证的最小集合）：

- `\\`：反斜杠
- `\"`：双引号
- `\n`：换行（LF）
- `\r`：回车（CR）
- `\t`：制表符（TAB）

说明：

- 解析阶段会把转义序列还原成真实字符（例如源码 `\"a\\n\"` 的值包含单个换行字符）。
- 后端生成 C 字符串字面量时会再次做 C 级别转义（例如把换行字符输出为 `\\n`），以保证生成的 `.c` 文件语法正确。
- 编译器与标准库测试里的源码 fixture 推荐优先使用 `"""..."""`，避免大量 `\n`/转义导致可读性下降。
- `"""..."""` 的 unindent 规则：
  - 先把换行规范化为 `\n`（`CRLF/CR` 都按 `LF` 处理）。
  - 如果开引号后紧跟换行，去掉这一行首空行。
  - 仅在“非空行”上计算最小共同缩进（空格数），并从每个非空行移除该缩进。
  - 若某行缩进使用 TAB，报错（当前约束：多行字符串缩进仅允许空格）。
  - 若闭引号独占一行，会去掉该闭引号前产生的尾部空行。

类型（设计草案 vs 当前实现）：

- 语言设计草案倾向将 `"..."` 视为 `&'static str`（可按 `&str` 使用）。
- Stage0/Stage1/Stage2 当前实现中，字符串字面量仍视为 `String`（后端以 `const char*` 表示）。
- Stage2 当前不支持裸 `str` 类型；请使用 `String`（拥有）或 `&str`/`&'static str`（借用）。
- Stage2 已支持 `&T` / `&mut T` / `&'static T` / `&'static mut T` 语法，但当前仍是过渡语义：类型检查阶段擦除为 `T`；命名 lifetime（如 `&'a T`）在语法阶段直接拒绝。
- 过渡到切片方向的当前落地是 `std/string` 的 `StrView { owner: String, lo, hi }`（拥有型视图），用于“可长期保存的子串视图”；推荐优先使用 view-first API（如 `sub`、`take_prefix`、`take_suffix`、`drop_prefix`、`drop_suffix`）并尽量延后 `to_string` 物化。

## 范围类型语法（已定，草案）

类型位置允许 `@range(lo..=hi) T`：

```vox
type Tiny = @range(0..=3) i32;
type Small = @range(-5..=5) i32;
```

范围类型在运行时的检查语义（已定）：

- 当发生到范围类型的显式转换时：
  - 编译期可确定越界：编译错误
  - 否则：运行时检查，失败 **panic**
- 若需要可恢复错误，请使用返回 `Option/Result` 的转换函数（例如 `Tiny::try_from(...)`）。

Stage0/Stage1 v0 当前实现限制：

- `T` 仅支持整数类型（当前 stage0/stage1 实现：`i8/u8/i16/u16/i32/u32/i64/u64/isize/usize`）。
- `lo/hi` 仅支持十进制整数字面量（允许前缀 `-`）。

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

## match 模式（patterns）

Stage0/Stage1 目前支持的 `match` pattern 形态：

- `_`：wildcard
- `name`：绑定模式（bind），总是匹配，并把 scrutinee 绑定到 `name`
- `true` / `false`：布尔字面量（仅当 scrutinee 是 `bool`）
- `123` / `-123`：整数字面量（仅当 scrutinee 是整数类型）
- `"txt"`：字符串字面量（仅当 scrutinee 是 `String`）
- `Enum.Variant(p0, p1, ...)`：枚举 variant pattern（payload 位置是 pattern，可递归）
- `.Variant(p0, p1, ...)`：枚举 variant pattern（点前缀简写；当枚举类型可由上下文确定时）

示例：

```vox
match x {
  .Some(0) => 0,
  .Some(v) => v,
  .None => -1,
}
```

递归 payload pattern：

```vox
match r {
  .Ok(.Some(v)) => v,
  .Ok(.None) => 0,
  .Err(_) => -1,
}
```

穷尽性（Stage0/Stage1 v0 的近似规则，已实现）：

- 若存在 `_` 或 `name` 这种“总是匹配”的 arm，则视为穷尽。
- 否则（enum scrutinee）：
  - unit variant（无 payload）：必须在某个 arm 中显式出现（例如 `.None => ...`）。
  - 单 payload variant：该 variant 的所有 arm 的 payload pattern 需要**联合**覆盖该 payload 类型；覆盖判定对 enum 类型可递归（对 int/String 等“无限域”类型则必须出现 `_`/绑定模式）。
    - 例如 `Result.Ok(Option[T])` 的 `.Ok(.Some(v))` 与 `.Ok(.None)` 组合起来即可覆盖 `.Ok(...)`。
  - 多 payload variant：仍需要一个该 variant 的 “catch-all arm”，其所有 payload pattern 都是 `_` 或绑定模式（例如 `.Pair(_, _)` / `.Pair(a, b)`）。
    - 允许同一个 variant 出现多个 arm：先写更具体的 payload pattern，再写该 variant 的 catch-all arm。
- 否则（非 enum scrutinee）：必须有 `_` 或绑定模式 arm（Stage0/Stage1 v0 不做完整穷尽推导）。

## 类型别名（type alias）

声明一个类型别名：

```vox
type I = i32;
type V = Vec[I];
```

别名本身只出现在**类型位置**（不引入值命名空间），例如不允许用 `Alias.Variant(...)` 作为 enum 构造子路径。

可以导出别名：

```vox
pub type Size = i64;
```

## 常量（const，Stage0/Stage1 v0）

声明一个模块级常量：

```vox
const N: i32 = 10;
pub const NAME: String = "vox";
```

约束（Stage0/Stage1 v0）：

- `const` 必须在顶层声明（不支持在函数内声明 const）。
- 必须写明类型注解：`const X: T = ...`
- 初始化表达式必须是 **const expression**（常量表达式），目前仅支持：
  - 字面量：`123` / `1.5` / `true` / `"txt"`
  - 其他常量引用（含跨模块的 `import` 访问）
  - 块表达式（最小子集）：`{ let x = ...; ...; tail }`
    - 支持 `let`/`let mut`、赋值语句（仅局部变量）、表达式语句、`if` 语句
    - `if` 语句按常量条件执行选中分支；分支内绑定不泄漏到外层
    - 已支持 `while`/`break`/`continue`/`return`（最小子集，语义与运行时一致）
    - 在“需要值”的上下文中必须有 tail 表达式（无 tail 视为不支持）
    - 在 unit 上下文（例如 `if` 表达式分支仅用于语句执行）允许省略 tail
  - `-x` / `!x`（`!` 对 `bool` 是逻辑非，对整数是按位非）
  - `expr as <int>`、`expr as f32/f64`（运行时与 const 场景均支持整数与浮点互转；`float -> int` 为 checked cast）
  - `+ - * / %`、`& | ^ << >>`、比较、`== !=`、`&& ||`
    - `&& ||` 在 const 中同样是短路求值：`false && rhs` / `true || rhs` 不会求值 `rhs`。
  - `if cond { a } else { b }`（cond 必须为常量 bool）
  - `match`（当前 const 子集支持 `_`、bind、`true/false`、整数字面量、字符串字面量、enum pattern（含 payload））
- 整数运算语义与运行时保持一致：
  - `+ - *`、`& | ^`、`<< >>` 按目标整数位宽执行（wrapping）。
  - `/ %` 在除数为 `0` 时编译期报错。
  - 有符号整数在 `MIN / -1` 与 `MIN % -1` 时编译期报错（`division overflow`）。
  - `<< >>` 的移位位数必须在 `[0, bit_width(T)-1]`，否则编译期报错。
  - 浮点常量（`f32/f64`）在 v0 支持：字面量、常量引用、`-x`、`f32 <-> f64 as`、`+ - * /`、`< <= > >=`、`== !=`。
  - `!x`：`bool` 上是逻辑非；整数上是按位非（按目标整数位宽 wrapping）。
  - 浮点常量的 `/` 在除数为 `0.0`（或规范化后为 `0.0`）时报错：`const division by zero`。
- 支持在 const 初始化中调用普通函数（当前子集）：
  - 支持非泛型与泛型函数调用。
  - 泛型调用语法与运行时一致：`f[T](...)`、`f[3](...)`、`f[T, 3](...)`（常量泛型参数使用 `@const` 语法路径传递）。
  - 类型泛型参数可由实参/期望类型推导，也可显式给出；常量泛型参数需显式给出。
  - 泛型 trait bound 在 const 调用中同样校验。
  - 形参/返回类型按普通类型检查规则校验，函数体按 const 语义执行。
  - 支持同模块函数与通过 `import` 可见的函数（遵循可见性规则）。
- 支持在 const 初始化中构造 struct 与读取字段，例如：
  - `const P0: P = P { x: 1, y: 2 }`
  - `const Q0: Pair[i32] = Pair[i32] { a: 1, b: 2 }`
  - `const X: i32 = P0.x`
- 支持在 const block 中对 `let mut` 的 struct 局部执行字段赋值（含多级路径与复合赋值，如 `p.x = ...`、`o.i.x = ...`、`o.i.x += 2`）。
- 支持 enum 变体构造（含 payload），例如 `const X: E = .A(1)`。
  - 也支持限定路径写法：`const X: E = E.A(1)`、`const X: dep.E = dep.E.A(1)`。
  - 也支持 typed-path 写法：`const X: Option[i32] = Option[i32].Some(1)`、`const Y: Option[i32] = Option[i32].None`。

可见性与导入（Stage0/Stage1）：

- 默认私有（仅当前模块可见）。
- 可见性修饰：
  - `pub`：对所有模块可见。
  - `pub(crate)`：对同一包（crate）内模块可见。
  - `pub(super)`：对父模块作用域可见（父模块及其子模块）。
- 对于 `src/**` 与 `tests/**`，当前实现把它们视为不同的包边界：`tests/**` 不能访问 `src/**` 的 `pub(crate)` 符号。
- 跨模块访问常量时需要目标符号对当前模块可见（例如 `pub const` 或受限 `pub(...)` 满足可见性规则）。
- 支持：
  - 通过模块别名访问：`import "pkg:dep" as d; d.MAX_RETRY`
  - 通过命名导入访问：`import { MAX_RETRY } from "pkg:dep"; MAX_RETRY`

### 绑定模式（bind pattern）

`match` 的 pattern 允许使用单个标识符作为“绑定模式”，它总是匹配，并将 scrutinee 绑定到该名字：

```vox
match x {
  v => match v {
    .Some(n) => n,
    .None => 0,
  },
}
```

说明：

- `v` 的类型等于 scrutinee 的类型（这里是 `Option[i32]`）。
- 绑定模式等价于“带名字的 `_`”，所以也会让 `match` 变为穷尽。
- Stage0/Stage1 v0 会对明显的 unreachable arm 报错（例如 `_`/绑定模式之后的 arm，或某个 enum variant 在 payload 空间已被覆盖之后的 arm）。

## 禁止的引用语法位置（目标语义）

为配合“临时借用”规则：

- 非 `&'static` 的 `&T` / `&mut T` 不允许出现在 struct 字段与返回类型中（详见 `docs/07-memory-management.md`）
- Stage2 当前为过渡实现，`&T` 在类型层先擦除为 `T`，所以暂未强制上述位置约束。

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
  - `v.insert(i: i32, x: T) -> ()`（可变；索引范围 `0..=len`，越界运行时 panic）
  - `v.set(i: i32, x: T) -> ()`（可变；索引范围 `0..len-1`，越界运行时 panic）
  - `v.clear() -> ()`（可变）
  - `v.extend(other: Vec[T]) -> ()`（可变）
  - `v.pop() -> T`（可变；空向量运行时 panic）
  - `v.remove(i: i32) -> T`（可变；索引越界运行时 panic）
  - `v.len() -> i32`
  - `v.is_empty() -> bool`
  - `v.get(i: i32) -> T`
  - `v.join(sep: String) -> String`（仅当 `T == String`；用于拼接字符串片段）
- `String`：
  - `s.len() -> i32`
  - `s.is_empty() -> bool`
  - `s.byte_at(i: i32) -> i32`
  - `s.slice(start: i32, end: i32) -> String`
  - `s.concat(other: String) -> String`（返回新字符串；用于代码生成等场景）
  - `s.starts_with(prefix: String) -> bool`
  - `s.ends_with(suffix: String) -> bool`
  - `s.contains(needle: String) -> bool`
  - `s.index_of(needle: String) -> i32`（未找到返回 `-1`；空 needle 返回 `0`）
  - `s.last_index_of(needle: String) -> i32`（未找到返回 `-1`；空 needle 返回 `s.len()`）
  - `s.escape_c() -> String`（返回可放入 C 字符串字面量的转义内容）
- `i32/i64/bool`：
  - `x.to_string() -> String`（最小格式化能力；用于诊断与代码生成）

对 receiver 的约束（Stage0）：

- 非变更方法（如 `len/is_empty/get/byte_at/slice/starts_with/ends_with/contains/index_of/last_index_of`）：receiver 可以是任意表达式（例如 `ctx.items.len()`）。
- 变更方法（`Vec.push`/`Vec.insert`/`Vec.set`/`Vec.clear`/`Vec.extend`/`Vec.pop`/`Vec.remove`）：receiver 必须是 **place**（可写位置）。
  - `Vec.push`/`Vec.insert`/`Vec.set`/`Vec.clear`/`Vec.extend`/`Vec.pop`/`Vec.remove` 要求 receiver 的根绑定是 `let mut`（不可变绑定会报错）。
  - Stage0/Stage1：支持局部变量（如 `v.push(x)`、`v.insert(i, x)`、`v.set(i, x)`、`v.clear()`、`v.extend(w)`、`v.pop()`、`v.remove(i)`），以及可变局部 struct 的直接字段（如 `s.items.push(x)`、`s.items.insert(i, x)`、`s.items.set(i, x)`、`s.items.clear()`、`s.items.extend(w)`、`s.items.pop()`、`s.items.remove(i)`）。
  - Stage2：额外支持可变局部 struct 的多级字段（如 `o.inner.items.push(x)`、`o.inner.items.insert(i, x)`、`o.inner.items.set(i, x)`、`o.inner.items.clear()`、`o.inner.items.extend(w)`、`o.inner.items.pop()`、`o.inner.items.remove(i)`）。

### 保留的 `__*` 低层 intrinsic（Stage0/Stage1 自举期）

除 `panic/print` 外，自举期还存在少量以 `__` 开头的低层 intrinsic（用于 `std/fs`、`std/process` 等最小工具链能力）。

约束（Stage0/Stage1）：

- **非 `std/**` 模块禁止直接调用**以 `__` 开头的函数（例如 `__exec(...)`）。
- 禁止用户代码定义以 `__` 开头的函数/类型名（保留给自举期 intrinsic 与标准库实现）。
- 这些名字保留给标准库实现与自举工具链，用户代码应通过 `std/fs`、`std/process` 等封装接口使用。

### 类型反射 intrinsic（Stage1 已实现）

当前可用：

- `@size_of(Type) -> usize`
- `@align_of(Type) -> usize`
- `@type(Type) -> TypeId`（Stage1 当前表示为 `usize`）
- `@type_name(Type) -> String`
- `@field_count(Type) -> usize`（当前支持 `struct/enum`）
- `@field_name(Type, I) -> String`（当前支持 `struct/enum`，`I` 为 const 索引）
- `@field_type(Type, I) -> String`（当前支持 `struct/enum`，`I` 为 const 索引）
- `@field_type_id(Type, I) -> TypeId`（当前支持 `struct` 字段与 `enum` variant；多 payload variant 返回稳定合成 `TypeId`）
- `@same_type(A, B) -> bool`
- `@assignable_to(Src, Dst) -> bool`
- `@castable_to(Src, Dst) -> bool`
- `@eq_comparable_with(A, B) -> bool`
- `@ordered_with(A, B) -> bool`
- `@same_layout(A, B) -> bool`
- `@bitcastable(A, B) -> bool`
- `@is_integer(Type) -> bool`
- `@is_signed_int(Type) -> bool`
- `@is_unsigned_int(Type) -> bool`
- `@is_float(Type) -> bool`
- `@is_bool(Type) -> bool`
- `@is_string(Type) -> bool`
- `@is_struct(Type) -> bool`
- `@is_enum(Type) -> bool`
- `@is_vec(Type) -> bool`
- `@is_range(Type) -> bool`
- `@is_eq_comparable(Type) -> bool`
- `@is_ordered(Type) -> bool`
- `@is_unit(Type) -> bool`
- `@is_numeric(Type) -> bool`
- `@is_zero_sized(Type) -> bool`

说明：

- 语法按 intrinsic 不同：
  - `@name(Type)`：`@size_of/@align_of/@type/@type_name/@field_count/@is_*`
  - `@name(Type, I)`：`@field_name/@field_type/@field_type_id`
  - `@name(A, B)`：`@same_type/@assignable_to/@castable_to/@eq_comparable_with/@ordered_with/@same_layout/@bitcastable`
  - 三种形态都允许尾逗号：`@is_integer(i32,)`、`@same_type(i32, i64,)`、`@field_name(S, 1,)`
- 类型位置额外支持：`@field_type(Type, I)`，例如 `type B = @field_type(S, 1)`（当前仅支持 `struct` 字段与 `enum` 的 unit/单 payload variant；多 payload variant 会拒绝）。
- 这些 intrinsic 会在 IR 生成时折叠为常量；在 `const` 上下文同样可用。
- `@size_of/@align_of` 采用 Stage1 当前 C 后端的目标布局模型。

## 导入语法（Stage0 最小子集）

模块/包导入：

```vox
import "dep"
import "dep" as d

// 显式消歧义：
import "pkg:dep" as d  // 依赖包 dep（vox.toml [dependencies]）
import "mod:dep" as d  // 本地模块 dep（src/dep/**）
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
- `import "x"` 的默认解析为“同包本地模块优先，其次依赖包”；若出现歧义，必须用 `pkg:` / `mod:` 显式指定。
