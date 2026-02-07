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

## panic

`panic` 用于不可恢复错误（断言失败、内部 bug 等）。是否支持 `panic=abort` 等策略见工具链文档。
