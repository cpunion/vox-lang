# Comptime 详解（草案）

## 确定性边界

编译期执行要求可复现：

- 允许：控制流、递归、循环、数组/结构体操作、纯计算、受控反射。
- 禁止（默认）：文件 IO、网络、系统时间、随机数、进程环境等非确定性来源。

后续若需要“受控编译期 IO”，应通过显式的构建系统接口完成（不在语言核心规范中隐式开放）。

## comptime 可执行性：推导 + 可选注解（已定）

Vox 不要求在函数上写 `comptime` 关键字。编译器在分析阶段推导“该函数是否可在编译期执行”。

执行规则（草案）：

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

## 反射内建（Stage1 已实现子集）

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
- 可用于普通表达式与 `const` 初始化（均会折叠为常量）。
- `@size_of/@align_of` 当前按 Stage1 C 后端目标布局模型计算。
- `@type` 返回编译期 `TypeId`（Stage1 当前表示为 `usize`）。
- `@type_name` 返回编译器的类型显示字符串。
- `@field_count` 当前支持 `struct/enum`（分别返回字段数/variant 数）。
- `@field_name` 当前支持 `struct/enum`（分别返回字段名/variant 名），`I` 为 `usize` const 实参。
- `@field_type` 当前支持 `struct/enum`（返回字段/variant payload 类型文本），`I` 为 `usize` const 实参。
- `@field_type_id` 当前支持 `struct` 字段与 `enum` 的 0/1 payload variant（多 payload variant 暂不支持）。
- `@assignable_to(Src, Dst)` 复用当前类型系统赋值规则（如 `@range` 到 base 的 widening）。
- `@castable_to(Src, Dst)` 复用当前 `as` 显式转换规则（int/float/range 相关）。
- `@eq_comparable_with(A, B)` 与 `==/!=` 二元规则对齐。
- `@ordered_with(A, B)` 与 `< <= > >=` 二元规则对齐。
- `@same_layout(A, B)` 判断两类型在当前 Stage1 布局模型下是否同尺寸且同对齐。
- `@bitcastable(A, B)` 当前与 `@same_layout` 等价（可按位重解释的最小判定）。
- `@is_*` 返回类型分类判定（当前要求 `T` 为 concrete type）。
- `@is_eq_comparable` 与 `==/!=` 能力对齐（含递归 struct/enum 字段检查）。
- `@is_ordered` 与 `< <= > >=` 能力对齐（当前 `int/float/string`）。
- `@is_unit` 判断是否为 `()`。
- `@is_numeric` 判断是否为数值类型（`int` 或 `float`，含 range）。
- `@is_zero_sized` 判断当前布局下 size 是否为 0（如 `()`）。

暂未实现（保留方向）：

```vox
@field_type(T, index: usize) -> type
```

## comptime 报错（Stage1 已实现）

```vox
@compile_error(msg: String)
```

当前语义：

- 参数必须是 1 个 `String` 表达式。
- 调用点直接触发编译错误，错误文本为 `compile_error: <msg>`。
- 该语义同样适用于 `const` 初始化中的调用。

## 资源限制

编译器可对 comptime 执行施加上限（步数/时间/内存），超限时报错并提示可能的无限递归/爆炸展开。
