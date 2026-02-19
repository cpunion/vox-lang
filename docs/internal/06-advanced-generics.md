# 高级泛型特性

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

当前实现（函数/trait 方法）：

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

当前实现（函数/trait 方法 + struct/enum 声明）：

- 语法：`where comptime N > 0, comptime N <= 256`
- 约束对象：
  - 已声明的 `const` 泛型参数（`comptime N < M`）
  - 已声明的类型参数布局反射（`comptime @size_of(T) <= 64`、`comptime @align_of(T) <= 8`）
  - 已声明的类型参数字段/variant 数反射（`comptime @field_count(T) <= 8`）
  - 已声明的类型参数 `TypeId` 反射（`comptime @type(T) > 0`）
- 右值：
  - 十进制整数字面量（支持负号）
  - `const` 参数（如 `comptime N < M`、`comptime @size_of(T) <= LIM`）
  - 反射项（如 `comptime @size_of(T) <= @align_of(U)`、`comptime @field_count(T) <= @type(U)`）
- 运算符：`== != < <= > >=`
- 校验时机：
  - 函数/trait 方法：调用点（含默认 const 实参）
  - generic struct/enum：类型实例化点（如 `Small[i32]`、`Tiny[i32]`）
- 默认值一致性：声明阶段会校验“默认 const 值是否满足 comptime where”（当约束涉及的参数都有默认值时）
- impl 一致性：`impl Trait for Type` 的方法必须与 trait 方法声明的 `comptime where` 约束一致

## 3. 泛型偏特化 / 专门化（当前实现最小可用）

目标：允许对同一 trait 的 impl 在“更具体类型”上覆盖通用实现，同时保持可判定性和稳定诊断。

当前实现（受控 specialization，接近 `min_specialization`）：

- 允许同一 trait 出现重叠 impl，但必须存在**严格特化**关系。
- 严格特化判定基于 impl 头部（`for` 类型 + 头部 type bounds + 头部 `where comptime` 约束）：
  - 先比较 `for` 类型偏序：`A` 比 `B` 更特化，当且仅当：`B` 可匹配 `A`，且 `A` 不能匹配所有 `B`。
  - 当 `for` 头等价时，再比较头部约束：
    - type bounds：覆盖更多约束（或更强超 trait 约束）的 impl 更特化；
    - `where comptime`：按约束集合覆盖比较（A 覆盖 B 且 B 不覆盖 A 时，A 更特化）。
  - 直观例子：`impl[T] Tag for T` 与 `impl[T: Eq] Tag for T`，后者更特化。
- 对同一接收者类型，分派选择“最特化且唯一”的 impl。
- 分派前会先检查 impl 头部约束（type bounds + `where comptime`）是否对当前接收者成立；不成立的候选不会参与竞争。
- 若重叠但不存在严格偏序（不可比较或等价重叠），编译期报错：
  - `overlapping impl without strict specialization: ...`
- 诊断增强（Stage2）：
  - 候选 impl 列表按稳定顺序输出（避免受声明/加载顺序影响）。
  - 歧义分派错误附带 `rank_trace`，展示候选头部比较结果（如 `incomparable` / `left more specific`）。

当前限制（后续可扩展）：

- 偏序不比较方法体语义；`where comptime` 当前采用“约束集合覆盖”比较，不做不等式语义蕴含推理（如 `<=8` 自动推导强于 `<=16`）。
- 仅在当前 `unify_ty` 支持的类型构造上参与判定（如 `Vec[T]` 场景）。

## 4. 可变参数泛型（Stage2 当前实现）

当前 Stage2 行为：

- parser/typecheck 接受类型参数 pack 声明语法：`fn zip[T...](...) -> ...`
- parser/typecheck 接受值参数 variadic 语法：`fn sum(xs: T...) -> ...`
- 参数降级规则：`xs: T...` 在收集阶段降级为 `xs: Vec[T]`，并保留 variadic 元信息用于调用点规则。
- 声明约束：
  - variadic 参数必须是最后一个参数，否则报错 `variadic parameter must be the last parameter`；
  - 当前只允许一个 type parameter pack，且必须位于类型参数列表末尾。

调用点（已实现双模式）：

- 泛型参数列表顺序规则：
  - 调用写作 `f[TypeArgs..., ConstArgs...]`，类型实参必须在前，const 实参必须在后；
  - 若顺序混用（例如 `f[3, i32]`），报错：`generic arg order error: type args must come before const args`。

- 打包调用（pack-call）：
  - `sum(1, 2, 3)` 会在调用侧自动构造 `Vec[T]` 作为最后一个实参；
  - `sum()`（无 variadic 实参）也合法，会传入空 `Vec[T]`。
- 显式 `Vec` 调用（vec-call）：
  - `let xs: Vec[i32] = Vec(); sum(xs);` 直接把最后一个实参当作已构造好的 `Vec[T]`。
- 双模式一致性：
  - 对 `fn take[T, const N: i32](head: T, tail: T...) -> T`，`take[i32, 3](7, 8, 9)`（pack-call）与 `take[i32, 3](7, xs)`（vec-call）均可通过。
- 固定参数 + variadic：
  - 对 `fn f(x: i32, ys: i32...)`，至少需要提供固定参数数量；不足时报
    `wrong number of args: expected at least N, got M`。

类型参数 pack 的当前语义：

- `T...`（type param pack 声明）已进入“可实质参与类型系统”的实现状态：
  - 支持单个尾部 type parameter pack；
  - 显式类型实参允许超过固定前缀，超出的 trailing 类型实参会绑定到 pack 名称；
  - 已支持异构 pack 的逐位置物化（per-position substitution）：
    - pack 参与参数/返回/variadic 元素/type bounds/`where comptime` 反射约束时，不再要求“同构绑定”；
    - 实例名会做稳定去歧义（如 `pack`, `pack__1`, ...），避免单态化冲突。
  - 已支持 pack 成员投影（`Pack.N`）参与物化。
  - 对无 pack 的普通泛型，仍保留 `expected at most N, got M` 的 arity 检查。

后续增强（非阻塞）：

- 基于 arity/shape 的进一步专门化优化与代码体积控制策略。

当前已落地的代码体积控制基线：

- 对需要 materialization 的 type-pack 调用增加 arity 上限（`16`），超出时报错：
  - `type pack arity exceeds materialization limit: <n> > 16`
