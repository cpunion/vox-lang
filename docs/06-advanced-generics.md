# 高级泛型特性（草案）

> 本章用于记录 Vox 的“能在编译期落地、可实现、可诊断”的高级泛型能力。未定部分明确标注。

## 1. Const 泛型

```vox
struct Buffer[T, const N: usize] {
  data: [T; N],
}
```

### const 泛型默认值（可选）

```vox
struct Buffer[T, const N: usize = 1024] {
  data: [T; N],
}
```

## 2. 编译期约束（comptime where）

```vox
fn process[T](x: T)
where
  comptime @size_of(T) <= 64
{
  // ...
}
```

```vox
struct SmallArray[T, const N: usize]
where
  comptime N > 0,
  comptime N <= 256
{
  data: [T; N],
}
```

## 3. 泛型偏特化 / 专门化（未定）

目标：允许在不引入通用 lifetime 系统的前提下，对特定类型/常量参数选择更优实现。

选项（待讨论）：

1. 受控 specialization（类似 Rust `min_specialization` 思路）：允许有限重叠 impl，并要求“更特化”关系可判定、无歧义。
2. comptime 分派：通过 `comptime if` 在泛型实现内部选择路径（可实现但可能影响可读性/可组合性）。
3. 仅允许“同名内在方法”的手写特化（不开放重叠 trait impl），减少一致性复杂度。

本章后续会补充：

- 重叠规则
- 选择优先级
- 与包升级/向后兼容的约束

## 4. 可变参数泛型（deferred）

保留方向：tuple/parameter pack 支持，优先级低于 const 泛型与 comptime 约束。
