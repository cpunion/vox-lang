# 泛型与编译期执行（comptime）

> 注：本章是“目标设计”。Stage0 只实现最小泛型子集（泛型函数 + 单态化），不包含 `comptime`/宏/trait 等。

## 设计目标

1. 编译期执行的表达力尽量接近普通代码（受确定性约束）。
2. 支持类型作为值、反射与生成，服务于“生成编译代码结构”的目标。

## comptime 函数与变量

```vox
fn factorial(n: i32) -> i32 {
  if n <= 1 { return 1; }
  n * factorial(n - 1)
}

const F10: i32 = factorial(10);
```

```vox
comptime let SIZE: usize = 1024 * 1024;
let buf: [u8; SIZE];
```

## comptime 控制流

```vox
fn process[T](x: T) -> T {
  comptime if @is_integer(T) {
    x
  } else {
    x
  }
}
```

```vox
fn sum[const N: usize](xs: [i32; N]) -> i32 {
  let mut s = 0;
  comptime for i in 0..N {
    s += xs[i];
  }
  s
}
```

## 类型作为值

```vox
fn size_of(T: type) -> usize { @size_of(T) }
const N: usize = size_of(i64);
```

## 与宏系统的关系

- `comptime` 用于“编译期计算/反射/选择分支/生成数据”。
- AST 宏用于“生成语法树/声明/表达式”等结构（见 `docs/10-macro-system.md`）。
