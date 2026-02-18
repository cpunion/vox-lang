# Comptime 详解

## 确定性边界

编译期执行要求可复现：

- 允许：控制流、递归、循环、数组/结构体操作、纯计算、受控反射。
- 禁止（默认）：文件 IO、网络、系统时间、随机数、进程环境等非确定性来源。

后续若需要“受控编译期 IO”，应通过显式的构建系统接口完成（不在语言核心规范中隐式开放）。

## comptime 可执行性：推导 + 可选注解（已定）

Vox 不要求在函数上写 `comptime` 关键字。编译器在分析阶段推导“该函数是否可在编译期执行”。

执行规则：

- 任何 `fn` 在满足“确定性边界”的前提下，都可能被编译期解释器执行。
- 当某次调用发生在编译期上下文（例如 `const` 初始化、`comptime {}`、宏展开）时：
  - 若被调用函数可 comptime 执行：在编译期执行。
  - 否则：编译错误。

可选注解：

```vox
@comptime
fn crc_table() -> [u32; 256] { ... }
```

含义：要求该函数必须满足 comptime 约束；若包含不允许的操作（IO/随机/系统时间等）则直接报错。

## 反射内建（已实现子集）

当前已实现：

```vox
@size_of(T) -> usize
@align_of(T) -> usize
@type(T) -> TypeId
@type_name(T) -> String
@field_count(T) -> usize
@field_name(T, I) -> String
@field_type(T, I) -> String
@field_type_id(T, I) -> TypeId
@same_type(A, B) -> bool
@assignable_to(Src, Dst) -> bool
@castable_to(Src, Dst) -> bool
@eq_comparable_with(A, B) -> bool
@ordered_with(A, B) -> bool
@same_layout(A, B) -> bool
@bitcastable(A, B) -> bool
@is_integer(T) -> bool
@is_signed_int(T) -> bool
@is_unsigned_int(T) -> bool
@is_float(T) -> bool
@is_bool(T) -> bool
@is_string(T) -> bool
@is_struct(T) -> bool
@is_enum(T) -> bool
@is_vec(T) -> bool
@is_range(T) -> bool
@is_eq_comparable(T) -> bool
@is_ordered(T) -> bool
@is_unit(T) -> bool
@is_numeric(T) -> bool
@is_zero_sized(T) -> bool
```

语法与约束（当前）：

- 调用形态按 intrinsic 不同：
  - `@name(Type)`：如 `@size_of/@align_of/@type/@type_name/@field_count/@is_*`
  - `@name(Type, I)`：`@field_name/@field_type/@field_type_id`（`I` 为 `usize` const 实参）
  - `@name(A, B)`：`@same_type/@assignable_to/@castable_to/@eq_comparable_with/@ordered_with/@same_layout/@bitcastable`
  - 以上三种形态均允许尾逗号（例如 `@is_integer(i32,)`、`@same_type(i32, i64,)`）。
- 可用于普通表达式与 `const` 初始化（均会折叠为常量）。
- `@size_of/@align_of` 当前按 C 后端目标布局模型计算。
- `@type` 返回编译期 `TypeId`（当前表示为 `usize`）。
- `@type_name` 返回编译器的类型显示字符串。
- `@field_count` 当前支持 `struct/enum`（分别返回字段数/variant 数）。
- `@field_name` 当前支持 `struct/enum`（分别返回字段名/variant 名），`I` 为 `usize` const 实参。
- `@field_type` 当前支持 `struct/enum`（返回字段/variant payload 类型文本），`I` 为 `usize` const 实参。
- `@field_type_id` 当前支持 `struct` 字段与 `enum` variant（含多 payload variant）。对多 payload variant，返回稳定的合成 `TypeId`（用于编译期比较与缓存键）。
- `@assignable_to(Src, Dst)` 复用当前类型系统赋值规则（如 `@range` 到 base 的 widening）。
- `@castable_to(Src, Dst)` 复用当前 `as` 显式转换规则（int/float/range 相关）。
- `@eq_comparable_with(A, B)` 与 `==/!=` 二元规则对齐。
- `@ordered_with(A, B)` 与 `< <= > >=` 二元规则对齐。
- `@same_layout(A, B)` 判断两类型在当前布局模型下是否同尺寸且同对齐。
- `@bitcastable(A, B)` 当前与 `@same_layout` 等价（可按位重解释的最小判定）。
- `@is_*` 返回类型分类判定（当前要求 `T` 为 concrete type）。
- `@is_eq_comparable` 与 `==/!=` 能力对齐（含递归 struct/enum 字段检查）。
- `@is_ordered` 与 `< <= > >=` 能力对齐（当前 `int/float/string`）。
- `@is_unit` 判断是否为 `()`。
- `@is_numeric` 判断是否为数值类型（`int` 或 `float`，含 range）。
- `@is_zero_sized` 判断当前布局下 size 是否为 0（如 `()`）。

类型位置反射（Stage2 已实现）：

```vox
type F0 = @field_type(S, 0)
```

当前规则：

- 仅允许在**类型位置**使用 `@field_type(T, I)`，返回字段/variant 对应的类型。
- `T` 必须是 concrete nominal（`struct` 或 `enum`，可含已实例化泛型实参）。
- `I` 必须是非负整数字面量（支持尾逗号：`@field_type(S, 1,)`）。
- `struct`：`I` 对应字段索引。
- `enum`：`I` 对应 variant 索引；unit variant 返回 `()`；单 payload variant 返回该 payload 类型。
- 多 payload enum variant 在当前阶段会被拒绝（尚无 tuple 类型语法承载该结果）。

## comptime 报错（已实现）

```vox
@compile_error(msg: String)
```

当前语义：

- 参数必须是 1 个 `String` 表达式。
- 允许尾逗号调用：`@compile_error("msg",)`。
- 调用点直接触发编译错误，错误文本为 `compile_error: <msg>`。
- 该语义同样适用于 `const` 初始化中的调用。

## const 上下文的方法调用子集（Stage2 已实现）

为保持“普通函数在编译期可执行”的体验，Stage2 的 const/comptime 执行器支持一组纯方法调用：

- `String`：
  - `len() -> i32`
  - `byte_at(i32) -> i32`
  - `slice(i32, i32) -> String`
  - `concat(String) -> String`
  - `escape_c() -> String`
  - `to_string() -> String`
- 基础类型：
  - `bool.to_string() -> String`
  - `int.to_string() -> String`
  - `float.to_string() -> String`（当前为稳定文本表示，后续可继续向运行时格式收敛）

边界说明：

- 当前只开放“确定性、无副作用”的方法。
- 越界类错误（如 `byte_at/slice`）会在编译期直接报错。
- 其他方法调用（尤其依赖运行时资源或未进入纯子集的方法）仍会在编译期被拒绝。

## 资源限制

编译器可对 comptime 执行施加上限（步数/时间/内存），超限时报错并提示可能的无限递归/爆炸展开。
