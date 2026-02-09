# 错误处理

## 核心类型

```vox
enum Result[T, E] { Ok(T), Err(E) }
enum Option[T] { Some(T), None }
```

## `?` 错误传播

```vox
fn read_config(path: &str) -> Result[String, IoError] {
  let f = File::open(path)?;
  let s = f.read_to_string()?;
  Ok(s)
}
```

展开等价（示意）：

```vox
let f = match File::open(path) {
  Ok(v) => v,
  Err(e) => return Err(e.into()),
};
```

Stage1 当前实现（v0）：

- `expr?` 已接入，语义上支持两种容器：
  - `Result[T, E]`：`Ok(v)` 继续，`Err(e)` 提前返回 `Err(e)`。
  - `Option[T]`：`Some(v)` 继续，`None` 提前返回 `None`。
- 约束：
  - `?` 只能用于 `Result/Option`。
  - 所在函数返回类型也必须是对应容器类型（`Result` 对 `Result`，`Option` 对 `Option`）。
  - `Result` 情况下，传播的 `Err` 类型需与函数返回的 `Err` 类型兼容（当前 v0 不做 `into()` 自动转换）。

## `try {}` 块

```vox
fn load() -> Result[Data, Error] {
  let r: Result[Data, Error] = try {
    let s = read_config("app.toml")?;
    parse(&s)?
  };
  r
}
```

Stage1 当前实现（v0）：

- `try { ... }` 已接入为表达式语法，具备块级传播语义：
  - 块内 `?` 失败时会提前结束 `try` 块并返回残差（`Err/None`），不会直接结束外层函数。
  - 正常结束时，尾表达式会自动包装为成功值（`Ok/Some`）。
  - 若尾表达式本身已是目标容器类型（`Result` 或 `Option`），则直接返回该值。

## panic

`panic` 用于不可恢复错误（断言失败、内部 bug 等）。是否支持 `panic=abort` 等策略见工具链文档。
