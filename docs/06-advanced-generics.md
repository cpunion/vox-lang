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

当前 Stage1 已实现（函数/trait 方法）：

- `fn addn[const N: i32 = 3](x: i32) -> i32 { ... }`
- 调用可省略有默认值的 const 实参：`addn(4)`
- 也可显式覆盖默认值：`addn[9](4)`
- 规则：一旦某个 const 参数声明了默认值，后续 const 参数也必须声明默认值（避免位置调用歧义）

## 2. 编译期约束（comptime where）

```vox
fn fit[T](x: T) -> i32
where
  comptime @size_of(T) <= 64,
  comptime @align_of(T) <= 8
{
  return 1;
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

当前 Stage1 已实现（函数/trait 方法）：

- 语法：`where comptime N > 0, comptime N <= 256`
- 约束对象：
  - 已声明的 `const` 泛型参数（`comptime N < M`）
  - 已声明的类型参数布局反射（`comptime @size_of(T) <= 64`、`comptime @align_of(T) <= 8`）
- 右值：十进制整数字面量（支持负号）或 `const` 参数（如 `comptime N < M`、`comptime @size_of(T) <= LIM`）
- 运算符：`== != < <= > >=`
- 校验时机：调用点（含默认 const 实参）
- 默认值一致性：声明阶段会校验“默认 const 值是否满足 comptime where”（当约束涉及的参数都有默认值时）
- impl 一致性：`impl Trait for Type` 的方法必须与 trait 方法声明的 `comptime where` 约束一致

## 3. 泛型偏特化 / 专门化（Stage1 最小可用）

目标：允许对同一 trait 的 impl 在“更具体类型”上覆盖通用实现，同时保持可判定性和稳定诊断。

当前 Stage1 已实现（受控 specialization，接近 `min_specialization`）：

- 允许同一 trait 出现重叠 impl，但必须存在**严格特化**关系。
- 严格特化判定基于 impl 头部 `for` 类型（当前以 `unify_ty` 可判定域为准）：
  - `A` 比 `B` 更特化，当且仅当：`B` 可匹配 `A`，且 `A` 不能匹配所有 `B`。
  - 直观例子：`impl[T] Tag for Vec[T]` 与 `impl Tag for Vec[i32]`，后者更特化。
- 对同一接收者类型，分派选择“最特化且唯一”的 impl。
- 若重叠但不存在严格偏序（不可比较或等价重叠），编译期报错：
  - `overlapping impl without strict specialization: ...`

当前限制（后续可扩展）：

- 偏序只看 impl 头部 `for` 类型，不比较方法体与 where 子句的语义强弱。
- 仅在当前 `unify_ty` 支持的类型构造上参与判定（如 `Vec[T]` 场景）。

## 4. 可变参数泛型（deferred）

保留方向：tuple/parameter pack 支持，优先级低于 const 泛型与 comptime 约束。
