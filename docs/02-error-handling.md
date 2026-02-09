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

- `expr?` 已接入，按语义糖降级为 `match` + 提前 `return .Err(e)`。
- 因此当前约束为：被传播值需要有 `Ok/Err` 变体；所在函数返回类型也需要兼容 `.Err(e)`。

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

- `try { ... }` 已接入为表达式语法入口，当前按块表达式处理（与 `{ ... }` 同构）。
- 在 `try` 块中配合 `?` 可直接使用；错误传播行为来自 `?` 的降级规则。

## panic

`panic` 用于不可恢复错误（断言失败、内部 bug 等）。是否支持 `panic=abort` 等策略见工具链文档。
