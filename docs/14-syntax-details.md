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

`import pub as where`

不安全：

`unsafe`

## 泛型语法

- 泛型：`f[T](...)`、`impl[T] ...`
- turbofish：`f::[T](...)`
- const 泛型：`const N: usize`

## 运算符优先级

沿用 Rust-like 的常见规则（后续可给出完整表）。

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

当枚举类型可由上下文确定时，允许用 `.Variant` 代替 `Enum::Variant`：

```vox
let x: Option[i32] = .Some(1);
let y: Result[i32, Err] = .Ok(1);

match x {
  .Some(v) => v,
  .None => 0,
}
```

上下文不足以确定枚举类型时，必须写全路径：`Option::Some(1)`。

## 禁止的引用语法位置

为配合“临时借用”规则：

- 非 `&'static` 的 `&T` / `&mut T` 不允许出现在 struct 字段与返回类型中（详见 `docs/07-memory-management.md`）
